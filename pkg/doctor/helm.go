package doctor

import (
	"fmt"
	"os"
	"os/exec"
)

// BuildControlCenterValues generates a temporary Helm values file that enables
// the Control Center ingress. Call only when controlCenterHost is non-empty.
// The caller is responsible for removing the returned file.
func BuildControlCenterValues(host string) (string, error) {
	content := fmt.Sprintf(`controlCenter:
  ingress:
    enabled: true
    hosts:
      - host: %s
        paths:
          - path: /
            pathType: Prefix
`, host)
	tmp, err := os.CreateTemp("", "orkestra-cc-values-*.yaml")
	if err != nil {
		return "", err
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

// InstallOrUpgradeOrkestra installs or upgrades the Orkestra Helm chart in an
// idempotent way. This function:
//
//  1. Ensures the Orkestra Helm repo exists and updates its index
//  2. Builds a complete `helm upgrade --install` command
//  3. Applies optional version constraints
//  4. Applies any number of values files (`-f file.yaml`)
//  5. Applies any additional Helm arguments (e.g. --set, --set-string, --atomic)
//
// The caller may pass arbitrary Helm flags through `args`, allowing full control
// over the installation behaviour while keeping Orkestra defaults intact.
func InstallOrUpgradeOrkestra(version string, valueFiles []string, args ...string) error {
	// Always add and update repo — add is idempotent, update ensures fresh index.
	_ = exec.Command("helm", "repo", "add", Orkestra, OrkestraChartRepo).Run()
	_ = exec.Command("helm", "repo", "update", Orkestra).Run()

	// Base Helm args — always upgrade --install (idempotent)
	helmArgs := []string{
		"upgrade", "--install",
		Orkestra,
		fmt.Sprintf("%s/%s", Orkestra, OrkestraChartName),
		"--namespace", OrkestraNamespace,
		"--create-namespace",
	}

	// Optional version
	if version != "" {
		helmArgs = append(helmArgs, "--version", version)
	}

	// Values files
	for _, f := range valueFiles {
		if f != "" {
			helmArgs = append(helmArgs, "-f", f)
		}
	}

	// Additional Helm args (e.g. --set foo=bar)
	helmArgs = append(helmArgs, args...)

	// Run Helm
	cmd := exec.Command("helm", helmArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing/upgrading Orkestra: %w", err)
	}

	return nil
}
