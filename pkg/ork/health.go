package ork

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/spinner"
)

const (
	healthCheckTimeout   = 200 * time.Second
	controlCenterDeploy  = OrkestraControlCenter
	runtimeLogDir        = "/tmp/orkestra"
	runtimeLogPath       = "/tmp/orkestra/runtime.log"
	controlCenterLogPath = "/tmp/orkestra/controlcenter.log"
)

// RuntimeStatus is the result of CheckRuntimeHealth.
type RuntimeStatus struct {
	Running bool
	Reason  string // set when Running is false
}

// OrkestraInstalled reports whether the Orkestra runtime Deployment exists
// in OrkestraNamespace.
func OrkestraInstalled() bool {
	out, err := exec.Command("kubectl", "get", "deploy",
		OrkestraRuntime,
		"-n", OrkestraNamespace,
		"--no-headers",
		"-o", "name",
	).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

// RuntimeDeployed reports whether the runtime Deployment exists, with a
// short timeout so it is safe to call during startup.
func RuntimeDeployed() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "get", "deploy", OrkestraRuntime,
		"-n", OrkestraNamespace, "--ignore-not-found", "-o", "name").Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

// CheckRuntimeHealth waits up to healthCheckTimeout for the Orkestra runtime
// Deployment to have at least one ready replica. It polls every 2 seconds.
// Returns immediately if pods are in CrashLoopBackOff.
func CheckRuntimeHealth() RuntimeStatus {
	spin := spinner.Start("  → Checking Orkestra runtime health...")

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			spin.Failure()
			return RuntimeStatus{
				Reason: fmt.Sprintf("timeout (%s) waiting for Orkestra runtime to become ready", healthCheckTimeout),
			}
		case <-ticker.C:
			status := checkRuntimeOnce(ctx)
			if status.Running {
				spin.Success()
				return status
			}
			if status.Reason != "no ready replicas" {
				spin.Failure()
				return status
			}
		}
	}
}

// FetchRuntimeLogs saves the last 100 log lines from the Orkestra runtime to
// /tmp/orkestra/runtime.log. If the Control Center Deployment exists but has
// no ready replicas, its logs are saved to /tmp/orkestra/controlcenter.log.
// Returns the last 10 lines of the runtime log for inline display.
func FetchRuntimeLogs() (tail string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if mkErr := os.MkdirAll(runtimeLogDir, 0755); mkErr != nil {
		return "", fmt.Errorf("creating log dir: %w", mkErr)
	}

	runtimeLogs := fetchDeployLogs(ctx, OrkestraRuntime, OrkestraNamespace, 100)
	if writeErr := os.WriteFile(runtimeLogPath, []byte(runtimeLogs), 0644); writeErr != nil {
		return "", fmt.Errorf("writing runtime log: %w", writeErr)
	}

	ccName, _ := exec.CommandContext(ctx, "kubectl", "get", "deploy", controlCenterDeploy,
		"-n", OrkestraNamespace, "--ignore-not-found", "-o", "name").Output()
	if len(bytes.TrimSpace(ccName)) > 0 {
		ccReady, _ := exec.CommandContext(ctx, "kubectl", "get", "deploy", controlCenterDeploy,
			"-n", OrkestraNamespace, "-o", `jsonpath={.status.readyReplicas}`).Output()
		if r := strings.TrimSpace(string(ccReady)); r == "" || r == "0" {
			ccLogs := fetchDeployLogs(ctx, controlCenterDeploy, OrkestraNamespace, 100)
			_ = os.WriteFile(controlCenterLogPath, []byte(ccLogs), 0644)
		}
	}

	lines := strings.Split(strings.TrimRight(runtimeLogs, "\n"), "\n")
	if start := len(lines) - 10; start > 0 {
		lines = lines[start:]
	}
	return strings.Join(lines, "\n"), nil
}

// SyncRuntime restarts the Orkestra runtime Deployment so it picks up a
// new Katalog ConfigMap. Waits for the rollout to complete (3 minute timeout).
func SyncRuntime() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	restart := exec.CommandContext(ctx, "kubectl", "rollout", "restart",
		"deploy/"+OrkestraRuntime, "-n", OrkestraNamespace)
	restart.Stdout = os.Stdout
	restart.Stderr = os.Stderr
	if err := restart.Run(); err != nil {
		return fmt.Errorf("restarting runtime: %w", err)
	}

	wait := exec.CommandContext(ctx, "kubectl", "rollout", "status",
		"deploy/"+OrkestraRuntime, "-n", OrkestraNamespace)
	wait.Stdout = os.Stdout
	wait.Stderr = os.Stderr
	if err := wait.Run(); err != nil {
		return fmt.Errorf("waiting for rollout: %w", err)
	}
	return nil
}

// KatalogChanged returns true when .orkestra/katalog.yaml has uncommitted
// changes or was touched by the most recent commit.
func KatalogChanged(dir string) bool {
	katalogPath := filepath.Join(".orkestra", "katalog.yaml")
	if out, err := exec.Command("git", "-C", dir, "diff", "HEAD", "--", katalogPath).Output(); err == nil {
		if len(bytes.TrimSpace(out)) > 0 {
			return true
		}
	}
	if out, err := exec.Command("git", "-C", dir, "diff", "HEAD~1", "HEAD", "--", katalogPath).Output(); err == nil {
		if len(bytes.TrimSpace(out)) > 0 {
			return true
		}
	}
	return false
}

// ── private helpers ───────────────────────────────────────────────────────────

func checkRuntimeOnce(ctx context.Context) RuntimeStatus {
	nameOut, err := exec.CommandContext(ctx,
		"kubectl", "get", "deploy", OrkestraRuntime,
		"-n", OrkestraNamespace, "--ignore-not-found", "-o", "name",
	).Output()
	if err != nil || len(bytes.TrimSpace(nameOut)) == 0 {
		return RuntimeStatus{Reason: "deployment " + OrkestraRuntime + " not found in " + OrkestraNamespace}
	}

	readyOut, _ := exec.CommandContext(ctx,
		"kubectl", "get", "deploy", OrkestraRuntime,
		"-n", OrkestraNamespace, "-o", `jsonpath={.status.readyReplicas}`,
	).Output()
	ready := strings.TrimSpace(string(readyOut))
	if ready == "" || ready == "0" {
		if reason := crashLoopReason(ctx); reason != "" {
			return RuntimeStatus{Reason: reason}
		}
		return RuntimeStatus{Reason: "no ready replicas"}
	}
	return RuntimeStatus{Running: true}
}

func crashLoopReason(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "kubectl", "get", "pods",
		"-n", OrkestraNamespace,
		"-o", `jsonpath={range .items[*]}{.metadata.name}{"\t"}{range .status.containerStatuses[*]}{.state.waiting.reason}{end}{"\n"}{end}`).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
		if len(parts) == 2 && strings.Contains(parts[0], "runtime") && parts[1] == "CrashLoopBackOff" {
			return fmt.Sprintf("pod %s is in CrashLoopBackOff", parts[0])
		}
	}
	return ""
}

func fetchDeployLogs(ctx context.Context, deploy, ns string, tailLines int) string {
	out, err := exec.CommandContext(ctx, "kubectl", "logs",
		"deploy/"+deploy, "-n", ns, fmt.Sprintf("--tail=%d", tailLines)).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("[failed to fetch logs for %s: %v]", deploy, err)
	}
	return string(out)
}
