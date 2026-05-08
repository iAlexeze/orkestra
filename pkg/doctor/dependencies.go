package doctor

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/orkspace/orkestra/pkg/spinner"
	"github.com/orkspace/orkestra/pkg/utils"
)

const (
	getHelm               = "curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash"
	kindIngressURL        = "https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml"
	kubernetesIngressURL  = "https://kubernetes.github.io/ingress-nginx"
	metricsInstallURL     = "https://github.com/kubernetes-sigs/metrics-server#installation"
	metricsHelmRepo       = "https://kubernetes-sigs.github.io/metrics-server/"
	getKubectl            = "curl -LO https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl && "
	makeKubectlExecutable = "chmod +x kubectl && sudo mv kubectl /usr/local/bin/"
)

// EnsureDependencies installs kubectl and helm if they are missing.
// Uses a spinner for each installation step.
func EnsureDependencies() error {
	if !KubectlAvailable() {
		spin := spinner.Start("  → Installing kubectl...")
		if err := installKubectl(); err != nil {
			spin.Failure()
			return fmt.Errorf("failed to install kubectl: %w", err)
		}
		spin.Success()
	}

	if !HelmAvailable() {
		spin := spinner.Start("  → Installing helm...")
		if err := installHelm(); err != nil {
			spin.Failure()
			return fmt.Errorf("failed to install helm: %w", err)
		}
		spin.Success()
	}

	return nil
}

func installKubectl() error {
	cmd := exec.Command("sh", "-c", getKubectl+makeKubectlExecutable)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func installHelm() error {
	cmd := exec.Command("sh", "-c", getHelm)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// EnsureIngressController installs nginx-ingress if no ingress controller is
// found on the current cluster. Uses the kind-specific manifest when the
// current context is a kind cluster; otherwise installs via Helm.
func EnsureIngressController() error {
	if detectIngressController() != IngressNone {
		return nil
	}

	fmt.Println("  → Installing ingress controller (nginx)...")

	contextOut, _ := exec.Command("kubectl", "config", "current-context").Output()
	isKind := strings.Contains(string(contextOut), "kind")

	var cmd *exec.Cmd
	if isKind {
		cmd = exec.Command("kubectl", "apply", "-f", kindIngressURL)
	} else {
		exec.Command("helm", "repo", "add", "ingress-nginx", kubernetesIngressURL).Run()
		exec.Command("helm", "repo", "update").Run()
		cmd = exec.Command("helm", "install", "ingress-nginx",
			"ingress-nginx/ingress-nginx",
			"--namespace", "ingress-nginx",
			"--create-namespace",
			"--wait", "--timeout=120s",
		)
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Printf("  %s Ingress controller ready\n", utils.SuccessMark())
	return nil
}

// EnsureMetricsServer ensures the Kubernetes metrics-server is installed.
//
// It checks for the metrics-server deployment using `kubectl get -o json`
// for unambiguous detection via exit status. If not present, it attempts
// installation via Helm, using a kind-specific flag when needed.
//
// Any failure (check or install) is treated as non-fatal: the error is logged
// and the user is instructed to install metrics-server manually.
func EnsureMetricsServer() error {
	// Check existence via exit code only
	checkCmd := exec.Command(
		"kubectl", "get", "deployment", "metrics-server",
		"-n", "kube-system",
		"-o", "json",
	)

	if err := checkCmd.Run(); err == nil {
		return nil // already installed
	}

	fmt.Println("  → Installing metrics-server...")

	// Detect current context
	ctxOut, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		fmt.Printf("  %s  Could not determine current context: %v\n", utils.WarningMark(), err)
		fmt.Printf("  %s Please install metrics-server manually: %s\n", utils.InfoMark(), metricsInstallURL)
		return nil
	}
	isKind := strings.Contains(string(ctxOut), "kind")

	// Add/update Helm repo
	if err := exec.Command(
		"helm", "repo", "add", "metrics-server", metricsHelmRepo,
	).Run(); err != nil {
		fmt.Printf("  %s  Failed to add Helm repo: %v\n", utils.WarningMark(), err)
		fmt.Printf("  %s Please install manually: %s\n", utils.InfoMark(), metricsInstallURL)
		return nil
	}

	if err := exec.Command("helm", "repo", "update").Run(); err != nil {
		fmt.Printf("  %s Failed to update Helm repos: %v\n", utils.WarningMark(), err)
		fmt.Printf("  %s Please install manually: %s\n", utils.InfoMark(), metricsInstallURL)
		return nil
	}

	// Build Helm args
	args := []string{
		"install", "metrics-server",
		"metrics-server/metrics-server",
		"--namespace", "kube-system",
		"--wait", "--timeout=120s",
	}

	if isKind {
		args = append(args, "--set", "args={--kubelet-insecure-tls}")
	}

	installCmd := exec.Command("helm", args...)
	installCmd.Stdout = io.Discard
	installCmd.Stderr = io.Discard

	if err := installCmd.Run(); err != nil {
		fmt.Printf("  %s  Failed to install metrics-server: %v\n", utils.WarningMark(), err)
		fmt.Printf("  %s Please install manually: %s\n", utils.InfoMark(), metricsInstallURL)
		return nil
	}

	// Verify installation
	verifyCmd := exec.Command(
		"kubectl", "get", "deployment", "metrics-server",
		"-n", "kube-system",
		"-o", "json",
	)

	if err := verifyCmd.Run(); err != nil {
		fmt.Printf("  %s  metrics-server installation could not be verified.", utils.WarningMark())
		fmt.Printf("  %s Please check or install manually: %s\n", utils.InfoMark(), metricsInstallURL)
		return nil
	}

	fmt.Printf("  %s Metrics server ready\n", utils.SuccessMark())
	return nil
}
