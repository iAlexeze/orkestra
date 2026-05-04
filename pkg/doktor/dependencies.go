package doktor

import (
	"fmt"
	"io"
	"os/exec"
)

func EnsureDependencies() error {
	if !KubectlAvailable() {
		if err := installKubectl(); err != nil {
			return fmt.Errorf("failed to install kubectl: %w", err)
		}
	}

	if !HelmAvailable() {
		if err := installHelm(); err != nil {
			return fmt.Errorf("failed to install helm: %w", err)
		}
	}

	return nil
}

func installKubectl() error {
	fmt.Println("  → Installing kubectl...")

	cmd := exec.Command("sh", "-c",
		"curl -LO https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl && "+
			"chmod +x kubectl && sudo mv kubectl /usr/local/bin/")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	return cmd.Run()
}

func installHelm() error {
	fmt.Println("  → Installing helm...")

	cmd := exec.Command("sh", "-c",
		"curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	return cmd.Run()
}
