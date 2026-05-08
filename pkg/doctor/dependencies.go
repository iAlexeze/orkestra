package doctor

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/orkspace/orkestra/pkg/spinner"
)

// EnsureDependencies installs kubectl and helm if they are missing.
// Uses a spinner for each installation step.
func EnsureDependencies() error {
	if !KubectlAvailable() {
		spin := spinner.Start("  → Installing kubectl...")
		if err := installKubectl(); err != nil {
			spin.WithFailure()
			return fmt.Errorf("failed to install kubectl: %w", err)
		}
		spin.WithSuccess()
	}

	if !HelmAvailable() {
		spin := spinner.Start("  → Installing helm...")
		if err := installHelm(); err != nil {
			spin.WithFailure()
			return fmt.Errorf("failed to install helm: %w", err)
		}
		spin.WithSuccess()
	}

	return nil
}

func installKubectl() error {
	cmd := exec.Command("sh", "-c",
		"curl -LO https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl && "+
			"chmod +x kubectl && sudo mv kubectl /usr/local/bin/")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func installHelm() error {
	cmd := exec.Command("sh", "-c",
		"curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}
