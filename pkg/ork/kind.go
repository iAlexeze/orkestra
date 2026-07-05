package ork

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/utils"
)

const (
	// KindClusterName is the default kind cluster name used by ork run --dev.
	KindClusterName    = "orkestra-playground"
	DefaultKindVersion = "v0.27.0"
)

// EnsureKindCluster creates a kind cluster named `name` if it does not already
// exist, then switches kubectl to its context. Downloads the kind binary from
// GitHub releases if not found in PATH or ~/.orkestra/bin.
// workers > 0 provisions that many worker nodes in addition to the control-plane.
// version selects the kind release to download; empty string uses DefaultKindVersion.
func EnsureKindCluster(name string, workers int, version string) error {
	kindBin, err := resolveKind(version)
	if err != nil {
		return err
	}

	out, _ := exec.Command(kindBin, "get", "clusters").Output()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == name {
			fmt.Printf("  ✓ Cluster '%s' already exists\n", name)
			return useKindContext(name)
		}
	}

	label := fmt.Sprintf("'%s'", name)
	if workers > 0 {
		label = fmt.Sprintf("'%s' (%d worker(s))", name, workers)
	}
	fmt.Printf("  → Creating local cluster %s...\n", label)

	args := []string{"create", "cluster", "--name", name}
	if workers > 0 {
		cfg, cfgErr := writeKindConfig(workers)
		if cfgErr != nil {
			return fmt.Errorf("kind config: %w", cfgErr)
		}
		defer os.Remove(cfg)
		args = append(args, "--config", cfg)
	}

	create := exec.Command(kindBin, args...)
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	if err := create.Run(); err != nil {
		return fmt.Errorf("kind create cluster: %w", err)
	}
	fmt.Printf("  %s Cluster '%s' ready\n", utils.SuccessMark(), name)

	spin := utils.StartSpinner("   → Waiting for nodes to be ready...")
	defer spin.Failure()
	if err := waitForNodesReady(5 * time.Minute); err != nil {
		return err
	}
	spin.Success()
	fmt.Printf("  %s Nodes ready\n", utils.SuccessMark())
	return useKindContext(name)
}

// DeleteKindCluster deletes a kind cluster by name. Safe to call when the
// cluster does not exist.
func DeleteKindCluster(name string) error {
	kindBin, err := resolveKind(DefaultKindVersion)
	if err != nil {
		return err
	}
	cmd := exec.Command(kindBin, "delete", "cluster", "--name", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ── private helpers ───────────────────────────────────────────────────────────

func useKindContext(name string) error {
	ctxName := "kind-" + name
	use := exec.Command("kubectl", "config", "use-context", ctxName)
	use.Stdout = os.Stdout
	use.Stderr = os.Stderr
	if err := use.Run(); err != nil {
		return fmt.Errorf("switching kubectl context to %s: %w", ctxName, err)
	}
	return nil
}

func resolveKind(version string) (string, error) {
	if version == "" {
		// No explicit version: prefer whatever is already in PATH.
		if p, err := exec.LookPath("kind"); err == nil {
			return p, nil
		}
		version = DefaultKindVersion
	}
	// Explicit version: use the versioned cache entry, downloading if needed.
	dest := filepath.Join(orkBinDir(), "kind-"+version)
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	return downloadKind(dest, version)
}

func downloadKind(dest, version string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	url := fmt.Sprintf(
		"https://github.com/kubernetes-sigs/kind/releases/download/%s/kind-%s-%s",
		version, goos, goarch,
	)

	fmt.Printf("  → Downloading kind %s...\n", version)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("kind: cannot create bin dir: %w", err)
	}

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("kind: download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("kind: download returned %s for %s", resp.Status, url)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", fmt.Errorf("kind: writing binary: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(dest)
		return "", fmt.Errorf("kind: writing binary: %w", err)
	}
	f.Close()

	fmt.Printf("  %s kind %s installed (%s)\n", utils.SuccessMark(), version, dest)
	return dest, nil
}

func writeKindConfig(workers int) (string, error) {
	f, err := os.CreateTemp("", "ork-kind-*.yaml")
	if err != nil {
		return "", err
	}
	defer f.Close()
	fmt.Fprintf(f, "kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n- role: control-plane\n")
	for range workers {
		fmt.Fprintf(f, "- role: worker\n")
	}
	return f.Name(), nil
}

func waitForNodesReady(timeout time.Duration) error {
	cmd := exec.Command("kubectl", "wait", "--for=condition=Ready",
		"node", "--all", fmt.Sprintf("--timeout=%ds", int(timeout.Seconds())))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nodes not ready after %s", timeout)
	}
	return nil
}

// orkBinDir returns ~/.orkestra/bin — where Orkestra stores downloaded tools.
func orkBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orkestra", "bin")
}
