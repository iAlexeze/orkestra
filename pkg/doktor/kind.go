package doktor

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// KindClusterName is the default kind cluster created by `ork deploy --dev`.
	KindClusterName = "orkestra-playground"
	kindVersion     = "v0.27.0"
)

// EnsureKindCluster creates a kind cluster named `name` if it does not already
// exist, then switches kubectl to its context. Downloads the kind binary from
// GitHub releases if not found in PATH or ~/.orkestra/bin — Go is not required.
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
	fmt.Printf("  ✓ Cluster '%s' ready\n", name)
	return useKindContext(name)
}

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

// resolveKind returns a path to the kind binary, downloading it if necessary.
// Go is not required — kind is a static binary downloaded from GitHub releases.
func resolveKind() (string, error) {
	if p, err := exec.LookPath("kind"); err == nil {
		return p, nil
	}

	dest := filepath.Join(orkestaBinDir(), "kind")
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	return downloadKind(dest)
}

func downloadKind(dest string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goarch == "amd64" {
		goarch = "amd64"
	} else if goarch == "arm64" {
		goarch = "arm64"
	}

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

// orkestaBinDir returns ~/.orkestra/bin — the directory where Orkestra stores
// downloaded tools (kind, kubectl, helm, cloudflared).
func orkestaBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orkestra", "bin")
}
