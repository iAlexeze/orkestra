package e2e

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// verifyExpectation polls until all conditions pass or timeout expires.
// workDir is the working directory for command checks — relative paths in
// commands and resource file refs resolve from there.
func verifyExpectation(ctx context.Context, exp orktypes.E2EExpectation, workDir string) error {
	timeout, err := time.ParseDuration(exp.Timeout)
	if err != nil || timeout <= 0 {
		timeout = 60 * time.Second
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		if err := checkAll(ctx, exp, workDir); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s waiting for %q: %w", timeout, exp.Name, checkAll(ctx, exp, workDir))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func checkAll(ctx context.Context, exp orktypes.E2EExpectation, workDir string) error {
	for _, r := range exp.Resources {
		if err := checkResource(ctx, r, workDir); err != nil {
			return err
		}
	}
	for _, cmd := range exp.Commands {
		if err := checkCommand(ctx, cmd, workDir); err != nil {
			return err
		}
	}
	return nil
}

// checkResource asserts the state of any Kubernetes resource using kubectl.
// Kind can be any built-in or custom resource kind.
func checkResource(ctx context.Context, r orktypes.E2EResourceCheck, workDir string) error {
	ns := r.Namespace
	if ns == "" {
		ns = "default"
	}

	// count=0: assert resource(s) do NOT exist.
	// A kubectl error (CRD unknown, type not registered) also means nothing exists — pass.
	if r.Count != nil && *r.Count == 0 {
		var args []string
		if r.Name != "" {
			args = []string{"get", r.Kind, r.Name, "-n", ns, "--ignore-not-found", "-o", "name"}
		} else {
			args = []string{"get", r.Kind, "-n", ns, "--ignore-not-found", "-o", "name"}
		}
		out, err := runKubectl(ctx, workDir, args...)
		if err != nil {
			return nil // kubectl error = CRD unknown or type missing = nothing exists
		}
		if strings.TrimSpace(out) != "" {
			if r.Name != "" {
				return fmt.Errorf("%s/%s still exists in %s", r.Kind, r.Name, ns)
			}
			return fmt.Errorf("%s still exists in namespace %s:\n%s", r.Kind, ns, out)
		}
		return nil
	}

	// Named resource: assert it exists.
	if r.Name != "" {
		out, err := runKubectl(ctx, workDir, "get", r.Kind, r.Name, "-n", ns, "--ignore-not-found", "-o", "name")
		if err != nil || strings.TrimSpace(out) == "" {
			return fmt.Errorf("%s/%s not found in %s", r.Kind, r.Name, ns)
		}
		if r.Ready {
			return checkReady(ctx, workDir, r.Kind, r.Name, ns)
		}
		return nil
	}

	// Unnamed: assert at least one (or exact count) exists.
	out, err := runKubectl(ctx, workDir, "get", r.Kind, "-n", ns, "-o", "name")
	if err != nil || strings.TrimSpace(out) == "" {
		return fmt.Errorf("no %s found in namespace %s", r.Kind, ns)
	}
	if r.Count != nil {
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != *r.Count {
			return fmt.Errorf("%s count in %s: want %d, got %d", r.Kind, ns, *r.Count, len(lines))
		}
	}
	if r.Ready {
		// Check readiness of the first result.
		first := strings.SplitN(strings.TrimSpace(out), "\n", 2)[0]
		if parts := strings.SplitN(first, "/", 2); len(parts) == 2 {
			first = parts[1]
		}
		return checkReady(ctx, workDir, r.Kind, first, ns)
	}
	return nil
}

// checkReady checks whether a resource has available replicas.
// Uses jsonpath to read status.availableReplicas — covers Deployments and
// StatefulSets. Returns nil (ready) if the field is absent (e.g. Services).
func checkReady(ctx context.Context, workDir, kind, name, ns string) error {
	out, err := runKubectl(ctx, workDir,
		"get", kind, name, "-n", ns, "-o", "jsonpath={.status.availableReplicas}")
	if err != nil {
		return fmt.Errorf("%s/%s in %s: not ready (%s)", kind, name, ns, strings.TrimSpace(out))
	}
	val := strings.TrimSpace(out)
	if val == "0" {
		return fmt.Errorf("%s/%s in %s: not ready (availableReplicas=0)", kind, name, ns)
	}
	// val == "" means field doesn't exist on this kind — treat as ready (e.g. Service).
	return nil
}

func checkCommand(ctx context.Context, c orktypes.E2ECommand, workDir string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", c.Run)
	if workDir != "" {
		cmd.Dir = workDir
	}
	out, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return fmt.Errorf("running %q: %w", c.Run, err)
		}
	}

	if exitCode != c.ExitCode {
		return fmt.Errorf("command %q: expected exit code %d, got %d\noutput: %s",
			c.Run, c.ExitCode, exitCode, strings.TrimSpace(string(out)))
	}
	if c.OutputContains != "" && !strings.Contains(string(out), c.OutputContains) {
		return fmt.Errorf("command %q: output does not contain %q\noutput: %s",
			c.Run, c.OutputContains, strings.TrimSpace(string(out)))
	}
	return nil
}

func runKubectl(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
