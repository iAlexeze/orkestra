package doktor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// KindClusterName is the default kind cluster created by `ork deploy --dev`.
	KindClusterName = "orkestra-playground"
	kindModule      = "sigs.k8s.io/kind@v0.31.0"
)

// EnsureKindCluster creates a kind cluster named `name` if it does not already
// exist, then switches kubectl to its context. It installs kind via
// `go install` when kind is not found in PATH or GOBIN.
func EnsureKindCluster(name string) error {
	kindBin, err := resolveKind()
	if err != nil {
		return err
	}

	// Check if the cluster already exists.
	out, _ := exec.Command(kindBin, "get", "clusters").Output()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == name {
			fmt.Printf("  Kind cluster '%s' already exists\n", name)
			return useKindContext(name)
		}
	}

	fmt.Printf("  Creating kind cluster '%s'...\n", name)
	create := exec.Command(kindBin, "create", "cluster", "--name", name)
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	if err := create.Run(); err != nil {
		return fmt.Errorf("kind create cluster: %w", err)
	}
	fmt.Printf("  ✓ Cluster '%s' ready\n", name)
	return useKindContext(name)
}

// useKindContext switches kubectl to the kind cluster's context.
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

// resolveKind returns a path to the kind binary, installing it if necessary.
func resolveKind() (string, error) {
	// kind already on PATH
	if p, err := exec.LookPath("kind"); err == nil {
		return p, nil
	}

	// kind installed in GOBIN / GOPATH/bin / ~/go/bin but not on PATH
	if p := kindInGobin(); p != "" {
		return p, nil
	}

	// Install via go install
	fmt.Printf("  Installing kind via: go install %s\n", kindModule)
	install := exec.Command("go", "install", kindModule)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("go install kind: %w", err)
	}

	if p := kindInGobin(); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("kind installed but binary not found in %s", gobinDir())
}

func gobinDir() string {
	if v := os.Getenv("GOBIN"); v != "" {
		return v
	}
	if v := os.Getenv("GOPATH"); v != "" {
		return filepath.Join(v, "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}

func kindInGobin() string {
	p := filepath.Join(gobinDir(), "kind")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
