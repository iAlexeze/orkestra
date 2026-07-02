package e2e

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	orkutils "github.com/orkspace/orkestra/pkg/utils"
)

const portForwardTimeout = 15 * time.Second

// verifyExpectation polls until all conditions pass or timeout expires.
// workDir is the working directory for command checks — relative paths in
// commands and resource file refs resolve from there.
func verifyExpectation(ctx context.Context, exp orktypes.E2EExpectation, workDir string) error {
	if exp.Wait != "" {
		if d, err := time.ParseDuration(exp.Wait); err == nil && d > 0 {
			select {
			case <-time.After(d):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

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
	// Commands run before resources so that action commands (e.g. cleanup deletes)
	// execute before the resource state is checked.
	for _, cmd := range exp.Commands {
		if err := checkCommand(ctx, cmd, workDir); err != nil {
			return err
		}
	}
	if exp.Kubectl != nil {
		if err := checkKubectl(ctx, exp.Kubectl, workDir); err != nil {
			return err
		}
	}
	for _, r := range exp.Resources {
		if err := checkResource(ctx, r, workDir); err != nil {
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
	if err := applyAssertions(string(out), assertions{OutputContains: c.OutputContains, OutputNotContains: c.OutputNotContains, GreaterThan: c.GreaterThan, LessThan: c.LessThan}); err != nil {
		return fmt.Errorf("command %q: %w\noutput: %s", c.Run, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// assertions holds all output assertion fields shared across e2e command and kubectl subcommand types.
type assertions struct {
	Equals            string
	NotEquals         string
	OutputContains    string
	OutputNotContains string
	GreaterThan       string
	LessThan          string
}

// applyAssertions checks all assertion fields against output.
// Empty fields are skipped. GreaterThan and LessThan parse the trimmed output
// as float64 — returns an error if the output is not numeric when either is set.
func applyAssertions(output string, a assertions) error {
	if a.OutputContains != "" && !strings.Contains(output, a.OutputContains) {
		return fmt.Errorf("output does not contain %q", a.OutputContains)
	}
	if a.OutputNotContains != "" && strings.Contains(output, a.OutputNotContains) {
		return fmt.Errorf("output must not contain %q", a.OutputNotContains)
	}
	trimmed := strings.TrimSpace(output)
	if a.Equals != "" && trimmed != a.Equals {
		return fmt.Errorf("output: want %q got %q", a.Equals, trimmed)
	}
	if a.NotEquals != "" && trimmed == a.NotEquals {
		return fmt.Errorf("output must not equal %q", a.NotEquals)
	}
	if a.GreaterThan != "" || a.LessThan != "" {
		got, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return fmt.Errorf("greaterThan/lessThan: output %q is not numeric", trimmed)
		}
		if a.GreaterThan != "" {
			want, err := strconv.ParseFloat(a.GreaterThan, 64)
			if err != nil {
				return fmt.Errorf("greaterThan: value %q is not numeric", a.GreaterThan)
			}
			if got <= want {
				return fmt.Errorf("output: want > %s, got %s", a.GreaterThan, trimmed)
			}
		}
		if a.LessThan != "" {
			want, err := strconv.ParseFloat(a.LessThan, 64)
			if err != nil {
				return fmt.Errorf("lessThan: value %q is not numeric", a.LessThan)
			}
			if got >= want {
				return fmt.Errorf("output: want < %s, got %s", a.LessThan, trimmed)
			}
		}
	}
	return nil
}

// checkKubectl runs all kubectl DSL subcommands in the block.
func checkKubectl(ctx context.Context, k *orktypes.E2EKubectl, workDir string) error {
	for i, e := range k.Apply {
		if err := checkKubectlApply(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.apply[%d]: %w", i, err)
		}
	}
	for i, e := range k.Patch {
		if err := checkKubectlPatch(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.patch[%d]: %w", i, err)
		}
	}
	for i, e := range k.Get {
		if err := checkKubectlGet(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.get[%d]: %w", i, err)
		}
	}
	for i, e := range k.Logs {
		if err := checkKubectlLogs(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.logs[%d]: %w", i, err)
		}
	}
	for i, e := range k.Describe {
		if err := checkKubectlDescribe(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.describe[%d]: %w", i, err)
		}
	}
	for i, e := range k.Exec {
		if err := checkKubectlExec(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.exec[%d]: %w", i, err)
		}
	}
	for i, e := range k.PortForward {
		if err := checkKubectlPortForward(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.port-forward[%d]: %w", i, err)
		}
	}
	for i, e := range k.Events {
		if err := checkKubectlEvents(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.events[%d]: %w", i, err)
		}
	}
	for i, e := range k.Auth {
		if err := checkKubectlAuth(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.auth[%d]: %w", i, err)
		}
	}
	for i, e := range k.Cp {
		if err := checkKubectlCp(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.cp[%d]: %w", i, err)
		}
	}
	for i, e := range k.Top {
		if err := checkKubectlTop(ctx, e, workDir); err != nil {
			return fmt.Errorf("kubectl.top[%d]: %w", i, err)
		}
	}
	return nil
}

func checkKubectlApply(ctx context.Context, e orktypes.E2EKubectlApply, workDir string) error {
	if e.Inline != "" {
		args := []string{"apply", "-f", "-"}
		if e.Namespace != "" {
			args = append(args, "-n", e.Namespace)
		}
		cmd := exec.CommandContext(ctx, "kubectl", args...)
		if workDir != "" {
			cmd.Dir = workDir
		}
		cmd.Stdin = strings.NewReader(e.Inline)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("kubectl apply (inline): %w\noutput: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	file := e.File
	if !filepath.IsAbs(file) && workDir != "" {
		file = filepath.Join(workDir, file)
	}
	args := []string{"apply", "-f", file}
	if e.Namespace != "" {
		args = append(args, "-n", e.Namespace)
	}
	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl apply -f %s: %s", e.File, out)
	}
	return nil
}

func checkKubectlPatch(ctx context.Context, e orktypes.E2EKubectlPatch, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}
	patchType := e.Type
	if patchType == "" {
		patchType = "merge"
	}
	args := []string{"patch", e.Kind, e.Name, "-n", ns, "--type", patchType, "-p", e.Patch}
	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl patch %s/%s: %s", e.Kind, e.Name, out)
	}
	return nil
}

func checkKubectlGet(ctx context.Context, e orktypes.E2EKubectlGet, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}

	var args []string
	if e.Field != "" {
		field := e.Field
		if !strings.HasPrefix(field, ".") {
			field = "." + field
		}
		args = []string{"get", e.Kind, e.Name, "-n", ns, "-o", "jsonpath={" + field + "}"}
	} else {
		format := e.Format
		if format == "" {
			format = "json"
		}
		args = []string{"get", e.Kind, e.Name, "-n", ns, "-o", format}
	}

	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl get %s/%s: %s", e.Kind, e.Name, out)
	}

	out, err = applyExtract(ctx, workDir, out, e.JQ, e.YQ)
	if err != nil {
		return fmt.Errorf("kubectl get %s/%s: %w", e.Kind, e.Name, err)
	}

	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl get %s/%s: %w\noutput: %s", e.Kind, e.Name, err, out)
	}
	return nil
}

func checkKubectlLogs(ctx context.Context, e orktypes.E2EKubectlLogs, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}

	args := []string{"logs", "-n", ns}
	if e.LabelSelector != "" {
		args = append(args, "-l", e.LabelSelector)
	} else {
		args = append(args, e.Name)
	}
	if e.Container != "" {
		args = append(args, "-c", e.Container)
	}
	if e.Since != "" {
		args = append(args, "--since="+e.Since)
	}

	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl logs: %s", out)
	}

	out, err = applyExtract(ctx, workDir, out, e.JQ, "")
	if err != nil {
		return fmt.Errorf("kubectl logs: %w", err)
	}

	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl logs: %w\noutput: %s", err, out)
	}
	return nil
}

func checkKubectlDescribe(ctx context.Context, e orktypes.E2EKubectlDescribe, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}

	args := []string{"describe", e.Kind, "-n", ns}
	if e.Name != "" {
		args = append(args, e.Name)
	} else if e.LabelSelector != "" {
		args = append(args, "-l", e.LabelSelector)
	}

	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl describe %s: %s", e.Kind, out)
	}

	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl describe %s: %w\noutput: %s", e.Kind, err, out)
	}
	return nil
}

func checkKubectlExec(ctx context.Context, e orktypes.E2EKubectlExec, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}

	pod := e.Name
	if pod == "" && e.LabelSelector != "" {
		var err error
		pod, err = runKubectl(ctx, workDir,
			"get", "pod", "-n", ns, "-l", e.LabelSelector,
			"-o", "jsonpath={.items[0].metadata.name}")
		if err != nil || strings.TrimSpace(pod) == "" {
			return fmt.Errorf("kubectl exec: no pod found for selector %q in %s", e.LabelSelector, ns)
		}
	}

	args := []string{"exec", "-n", ns, pod}
	if e.Container != "" {
		args = append(args, "-c", e.Container)
	}
	args = append(args, "--")
	args = append(args, e.Command...)

	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl exec %s: %s", pod, out)
	}

	out, err = applyExtract(ctx, workDir, out, e.JQ, e.YQ)
	if err != nil {
		return fmt.Errorf("kubectl exec %s: %w", pod, err)
	}

	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl exec %s: %w\noutput: %s", pod, err, out)
	}
	return nil
}

func checkKubectlPortForward(ctx context.Context, e orktypes.E2EKubectlPortForward, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}

	var target string
	if e.LeaderElection != nil {
		leaseNs := e.LeaderElection.Namespace
		if leaseNs == "" {
			leaseNs = ns
		}
		holder, err := runKubectl(ctx, workDir,
			"get", "lease", e.LeaderElection.Lease, "-n", leaseNs,
			"-o", "jsonpath={.spec.holderIdentity}")
		if err != nil || strings.TrimSpace(holder) == "" {
			return fmt.Errorf("kubectl port-forward: lease %s/%s has no holder yet", leaseNs, e.LeaderElection.Lease)
		}
		target = "pod/" + strings.TrimSpace(holder)
	} else if e.Service != "" {
		target = "svc/" + e.Service
	} else {
		target = e.Pod
	}

	return doPortForwardCurl(ctx, e, ns, target, workDir)
}

// doPortForwardCurl opens a port-forward, optionally waits, runs curl, extracts,
// and asserts the response in one shot.
func doPortForwardCurl(ctx context.Context, e orktypes.E2EKubectlPortForward, ns, target, workDir string) error {
	raw, err := doPortForwardCurlRaw(ctx, e, ns, target, workDir)
	if err != nil {
		return err
	}
	return assertPortForwardOutput(ctx, workDir, raw, e, target)
}

// doPortForwardCurlRaw opens a port-forward and returns the raw curl output.
func doPortForwardCurlRaw(ctx context.Context, e orktypes.E2EKubectlPortForward, ns, target, workDir string) (string, error) {
	localPort, err := freeLocalPort()
	if err != nil {
		return "", fmt.Errorf("kubectl port-forward %s: could not find free local port: %w", target, err)
	}
	pfArgs := []string{"port-forward", "-n", ns, target,
		fmt.Sprintf("%s:%d", localPort, e.Port)}

	pfCmd := exec.CommandContext(ctx, "kubectl", pfArgs...)
	if workDir != "" {
		pfCmd.Dir = workDir
	}
	if err := pfCmd.Start(); err != nil {
		return "", fmt.Errorf("kubectl port-forward %s: %w", target, err)
	}
	defer func() { _ = pfCmd.Process.Kill() }()

	// Poll until port is open (max 15s).
	addr := "localhost:" + localPort
	deadline := time.Now().Add(portForwardTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	if e.Path == "" {
		return "", nil
	}

	if e.Wait != "" {
		if d, err := time.ParseDuration(e.Wait); err == nil && d > 0 {
			select {
			case <-time.After(d):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	method := e.Method
	if method == "" {
		method = "GET"
	}
	url := fmt.Sprintf("http://localhost:%s%s", localPort, e.Path)
	curlArgs := []string{"-s", "-X", method, url}

	sp := orkutils.StartSpinner(fmt.Sprintf("curl %s (→ %s)", url, target))
	curlCmd := exec.CommandContext(ctx, "curl", curlArgs...)
	if workDir != "" {
		curlCmd.Dir = workDir
	}
	curlOut, err := curlCmd.CombinedOutput()
	sp.Stop()
	if err != nil {
		return "", fmt.Errorf("curl %s: %w\noutput: %s", url, err, strings.TrimSpace(string(curlOut)))
	}
	return strings.TrimSpace(string(curlOut)), nil
}

// assertPortForwardOutput applies jq/yq extraction and assertions to raw curl output.
func assertPortForwardOutput(ctx context.Context, workDir, raw string, e orktypes.E2EKubectlPortForward, target string) error {
	out, err := applyExtract(ctx, workDir, raw, e.JQ, e.YQ)
	if err != nil {
		return fmt.Errorf("kubectl port-forward %s%s: %w", target, e.Path, err)
	}

	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl port-forward %s%s: %w\noutput: %s", target, e.Path, err, out)
	}
	return nil
}

func checkKubectlEvents(ctx context.Context, e orktypes.E2EKubectlEvents, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}
	args := []string{"events", "--for", e.Kind + "/" + e.Name, "-n", ns}
	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl events %s/%s: %s", e.Kind, e.Name, out)
	}
	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl events %s/%s: %w\noutput: %s", e.Kind, e.Name, err, out)
	}
	return nil
}

func checkKubectlAuth(ctx context.Context, e orktypes.E2EKubectlAuth, workDir string) error {
	args := []string{"auth", "can-i", e.Verb, e.Resource}
	if e.Namespace != "" {
		args = append(args, "-n", e.Namespace)
	}
	if e.As != "" {
		args = append(args, "--as", e.As)
	}
	// can-i exits 1 when the answer is "no" — treat as non-error so assertions decide.
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	raw, _ := cmd.CombinedOutput()
	out := strings.TrimSpace(string(raw))
	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl auth can-i %s %s: %w\noutput: %s", e.Verb, e.Resource, err, out)
	}
	return nil
}

func checkKubectlCp(ctx context.Context, e orktypes.E2EKubectlCp, workDir string) error {
	ns := e.Namespace
	if ns == "" {
		ns = "default"
	}

	pod := e.Name
	if pod == "" && e.LabelSelector != "" {
		var err error
		pod, err = runKubectl(ctx, workDir,
			"get", "pod", "-n", ns, "-l", e.LabelSelector,
			"-o", "jsonpath={.items[0].metadata.name}")
		if err != nil || strings.TrimSpace(pod) == "" {
			return fmt.Errorf("kubectl cp: no pod found for selector %q in %s", e.LabelSelector, ns)
		}
	}

	tmp, err := os.CreateTemp("", "ork-e2e-cp-*")
	if err != nil {
		return fmt.Errorf("kubectl cp: creating temp file: %w", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	src := fmt.Sprintf("%s/%s:%s", ns, pod, e.Src)
	args := []string{"cp", src, tmp.Name()}
	if e.Container != "" {
		args = append(args, "-c", e.Container)
	}
	if out, err := runKubectl(ctx, workDir, args...); err != nil {
		return fmt.Errorf("kubectl cp %s: %s", src, out)
	}

	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		return fmt.Errorf("kubectl cp: reading temp file: %w", err)
	}
	out := string(raw)

	out, err = applyExtract(ctx, workDir, out, e.JQ, e.YQ)
	if err != nil {
		return fmt.Errorf("kubectl cp %s: %w", src, err)
	}

	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl cp %s: %w\noutput: %s", src, err, out)
	}
	return nil
}

func checkKubectlTop(ctx context.Context, e orktypes.E2EKubectlTop, workDir string) error {
	kind := strings.ToLower(e.Kind)
	args := []string{"top", kind}
	if kind == "pod" || kind == "pods" {
		ns := e.Namespace
		if ns == "" {
			ns = "default"
		}
		args = append(args, "-n", ns)
		if e.LabelSelector != "" {
			args = append(args, "-l", e.LabelSelector)
		} else if e.Name != "" {
			args = append(args, e.Name)
		}
		if e.Containers {
			args = append(args, "--containers")
		}
	} else if e.Name != "" {
		args = append(args, e.Name)
	}
	out, err := runKubectl(ctx, workDir, args...)
	if err != nil {
		return fmt.Errorf("kubectl top %s: %s", kind, out)
	}
	if err := applyAssertions(out, assertions{Equals: e.Equals, NotEquals: e.NotEquals, OutputContains: e.OutputContains, OutputNotContains: e.OutputNotContains, GreaterThan: e.GreaterThan, LessThan: e.LessThan}); err != nil {
		return fmt.Errorf("kubectl top %s: %w\noutput: %s", kind, err, out)
	}
	return nil
}

// freeLocalPort finds an available local TCP port by binding to :0 and releasing it.
func freeLocalPort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	return port, err
}

// applyExtract pipes output through jq or yq if set.
func applyExtract(ctx context.Context, workDir, output, jq, yq string) (string, error) {
	if jq != "" {
		if !strings.HasPrefix(jq, ".") {
			jq = "." + jq
		}
		cmd := exec.CommandContext(ctx, "jq", "-r", jq)
		if workDir != "" {
			cmd.Dir = workDir
		}
		cmd.Stdin = strings.NewReader(output)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("jq %q: %w\ninput: %s", jq, err, output)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if yq != "" {
		cmd := exec.CommandContext(ctx, "yq", "e", yq, "-")
		if workDir != "" {
			cmd.Dir = workDir
		}
		cmd.Stdin = strings.NewReader(output)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("yq %q: %w\ninput: %s", yq, err, output)
		}
		return strings.TrimSpace(string(out)), nil
	}
	return output, nil
}

func runKubectl(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
