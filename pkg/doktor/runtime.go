package doktor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	healthCheckTimeout   = 200 * time.Second
	controlCenterDeploy  = "orkestra-cc"
	runtimeLogDir        = "/tmp/orkestra"
	runtimeLogPath       = "/tmp/orkestra/runtime.log"
	controlCenterLogPath = "/tmp/orkestra/controlcenter.log"
)

// RuntimeStatus is the result of CheckRuntimeHealth.
type RuntimeStatus struct {
	Running bool
	Reason  string // set when Running is false
}

// CheckRuntimeHealth reports whether the Orkestra runtime deployment is up
// and has at least one ready replica. Pods in CrashLoopBackOff are treated
// as not running and their pod name is included in Reason.
func CheckRuntimeHealth() RuntimeStatus {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	// Verify the deployment exists.
	nameOut, err := exec.CommandContext(ctx, "kubectl", "get", "deploy", OrkestraRuntime,
		"-n", OrkestraNamespace, "--ignore-not-found", "-o", "name").Output()
	if err != nil || len(bytes.TrimSpace(nameOut)) == 0 {
		return RuntimeStatus{Reason: "deployment " + OrkestraRuntime + " not found in " + OrkestraNamespace}
	}

	// Check ready replicas — Kubernetes omits the field when zero.
	readyOut, _ := exec.CommandContext(ctx, "kubectl", "get", "deploy", OrkestraRuntime,
		"-n", OrkestraNamespace, "-o", `jsonpath={.status.readyReplicas}`).Output()
	ready := strings.TrimSpace(string(readyOut))
	if ready == "" || ready == "0" {
		if reason := crashLoopReason(ctx); reason != "" {
			return RuntimeStatus{Reason: reason}
		}
		return RuntimeStatus{Reason: "no ready replicas"}
	}

	return RuntimeStatus{Running: true}
}

// crashLoopReason scans pods in orkestra-system for any runtime pod in
// CrashLoopBackOff and returns a human-readable reason string.
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

// FetchRuntimeLogs saves the last 100 log lines from the Orkestra runtime to
// /tmp/orkestra/runtime.log. If the control-center deployment exists but has
// no ready replicas, its logs are also saved to /tmp/orkestra/controlcenter.log.
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

	// Control center: best-effort, captured only when it exists but is unhealthy.
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

func fetchDeployLogs(ctx context.Context, deploy, ns string, tailLines int) string {
	out, err := exec.CommandContext(ctx, "kubectl", "logs",
		"deploy/"+deploy, "-n", ns, fmt.Sprintf("--tail=%d", tailLines)).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("[failed to fetch logs for %s: %v]", deploy, err)
	}
	return string(out)
}

// KatalogChanged returns true when .orkestra/katalog.yaml has uncommitted
// changes or was modified by the most recent commit. Either condition means
// the operator should be restarted after the bundle is applied so it picks
// up the new configuration.
func KatalogChanged(dir string) bool {
	katalogPath := filepath.Join(".orkestra", "katalog.yaml")

	// Uncommitted changes — working tree or index vs HEAD.
	if out, err := exec.Command("git", "-C", dir, "diff", "HEAD", "--", katalogPath).Output(); err == nil {
		if len(bytes.TrimSpace(out)) > 0 {
			return true
		}
	}

	// Last commit touched it. Silently ignored on a shallow repo or first commit.
	if out, err := exec.Command("git", "-C", dir, "diff", "HEAD~1", "HEAD", "--", katalogPath).Output(); err == nil {
		if len(bytes.TrimSpace(out)) > 0 {
			return true
		}
	}

	return false
}

// RestartOrkestra issues a rollout restart of the runtime deployment and
// waits up to 3 minutes for it to become available again.
func RestartOrkestra() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	restart := exec.CommandContext(ctx, "kubectl", "rollout", "restart",
		"deploy/"+OrkestraRuntime, "-n", OrkestraNamespace)
	restart.Stdout = os.Stdout
	restart.Stderr = os.Stderr
	if err := restart.Run(); err != nil {
		return fmt.Errorf("rollout restart: %w", err)
	}

	wait := exec.CommandContext(ctx, "kubectl", "rollout", "status",
		"deploy/"+OrkestraRuntime, "-n", OrkestraNamespace)
	wait.Stdout = os.Stdout
	wait.Stderr = os.Stderr
	return wait.Run()
}
