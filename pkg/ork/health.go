package ork

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	healthCheckTimeout    = 200 * time.Second
	resourceExistsTimeout = 10 * time.Second
	controlCenterDeploy   = OrkestraControlCenter
	runtimeLogDir         = "/tmp/orkestra"
	runtimeLogPath        = "/tmp/orkestra/runtime.log"
	gatewayLogDir         = "/tmp/orkestra"
	gatewayLogPath        = "/tmp/orkestra/gateway.log"
	controlCenterLogPath  = "/tmp/orkestra/controlcenter.log"
)

// RuntimeInstalled reports whether the Orkestra runtime Deployment exists.
func RuntimeInstalled() bool {
	return ResourceExists("deploy", OrkestraRuntime, OrkestraNamespace)
}

// GatewayInstalled reports whether the Orkestra gateway Deployment exists.
func GatewayInstalled() bool {
	return ResourceExists("deploy", OrkestraGateway, OrkestraNamespace)
}

// CheckRuntimeHealth waits up to healthCheckTimeout for the Orkestra runtime
// Deployment to have at least one ready replica. It polls every 2 seconds.
// Returns immediately if pods are in CrashLoopBackOff.
var (
	runtimeChecker       = DeploymentHealthChecker{Name: OrkestraRuntime, Namespace: OrkestraNamespace}
	gatewayChecker       = DeploymentHealthChecker{Name: OrkestraGateway, Namespace: OrkestraNamespace}
	controlCenterChecker = DeploymentHealthChecker{Name: controlCenterDeploy, Namespace: OrkestraNamespace}
)

// CheckRuntimeHealth waits up to healthCheckTimeout for the Orkestra runtime to be ready
func CheckRuntimeHealth() DeploymentStatus {
	return runtimeChecker.CheckHealth(healthCheckTimeout, func(ctx context.Context) string {
		return crashLoopReason(ctx)
	})
}

// CheckGatewayHealth waits up to healthCheckTimeout for the Orkestra gateway to be ready
func CheckGatewayHealth() DeploymentStatus {
	return gatewayChecker.CheckHealth(healthCheckTimeout, nil)
}

// FetchGatewayLogs saves gateway logs and returns the last 10 lines
func FetchGatewayLogs() (tail string, err error) {
	return gatewayChecker.FetchLogs(100, gatewayLogDir, gatewayLogPath)
}

// FetchControlCenterLogsIfNeeded fetches control center logs only if the deployment exists but has no ready replicas
func FetchControlCenterLogsIfNeeded() error {
	if !controlCenterChecker.Exists() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !controlCenterChecker.HasReadyReplicas(ctx) {
		_, err := controlCenterChecker.FetchLogs(100, runtimeLogDir, controlCenterLogPath)
		return err
	}
	return nil
}

// FetchRuntimeLogs saves the last 100 log lines from the Orkestra runtime to
// /tmp/orkestra/runtime.log. If the Control Center Deployment exists but has
// no ready replicas, its logs are saved to /tmp/orkestra/controlcenter.log.
// Returns the last 10 lines of the runtime log for inline display.
func FetchRuntimeLogs() (tail string, err error) {
	tail, err = runtimeChecker.FetchLogs(100, runtimeLogDir, runtimeLogPath)
	if err != nil {
		return "", err
	}

	// Optionally fetch control center logs if needed
	_ = FetchControlCenterLogsIfNeeded()

	return tail, nil
}

// SyncRuntime restarts the Orkestra runtime Deployment and waits for rollout.
func SyncRuntime() error {
	return SyncDeployment(OrkestraRuntime, OrkestraNamespace, 3*time.Minute)
}

// SyncGateway restarts the Orkestra gateway Deployment and waits for rollout.
func SyncGateway() error {
	return SyncDeployment(OrkestraGateway, OrkestraNamespace, 3*time.Minute)
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
