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

	"github.com/orkspace/orkestra/pkg/spinner"
	"github.com/orkspace/orkestra/pkg/utils"
)

const (
	// KindClusterName is the default kind cluster name used by ork run --dev.
	KindClusterName = "orkestra-playground"
	kindVersion     = "v0.27.0"
)

// EnsureKindCluster creates a kind cluster named `name` if it does not already
// exist, then switches kubectl to its context. Downloads the kind binary from
// GitHub releases if not found in PATH or ~/.orkestra/bin.
func EnsureKindCluster(name string) error {
	kindBin, err := resolveKind()
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

	fmt.Printf("  → Creating local cluster '%s'...\n", name)
	create := exec.Command(kindBin, "create", "cluster", "--name", name)
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	if err := create.Run(); err != nil {
		return fmt.Errorf("kind create cluster: %w", err)
	}
	fmt.Printf("  %s Cluster '%s' ready\n", utils.SuccessMark(), name)

	spin := spinner.Start("   → Waiting for nodes to be ready...")
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
	kindBin, err := resolveKind()
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

func resolveKind() (string, error) {
	if p, err := exec.LookPath("kind"); err == nil {
		return p, nil
	}
	dest := filepath.Join(orkBinDir(), "kind")
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	return downloadKind(dest)
}

func downloadKind(dest string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	url := fmt.Sprintf(
		"https://github.com/kubernetes-sigs/kind/releases/download/%s/kind-%s-%s",
		kindVersion, goos, goarch,
	)

	fmt.Printf("  → Downloading kind %s...\n", kindVersion)
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

	fmt.Printf("  ✓ kind %s installed (%s)\n", kindVersion, dest)
	return dest, nil
}

func waitForNodesReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("kubectl", "get", "nodes",
			"--no-headers", "-o", "custom-columns=STATUS:.status.conditions[-1].type").Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			allReady := len(lines) > 0
			for _, l := range lines {
				if strings.TrimSpace(l) != "Ready" {
					allReady = false
					break
				}
			}
			if allReady {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("nodes not ready after %s", timeout)
}

// orkBinDir returns ~/.orkestra/bin — where Orkestra stores downloaded tools.
func orkBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orkestra", "bin")
}
