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

func InstallOrUpgradeOrkestra(version string, valueFiles []string, upgrade bool) error {
	// Always add and update repo — add is idempotent, update ensures fresh index.
	repoAdd := exec.Command("helm", "repo", "add", Orkestra, OrkestraChartRepo)
	repoAdd.Stdout = os.Stdout
	repoAdd.Stderr = os.Stderr
	_ = repoAdd.Run() // ignore "already exists"

	update := exec.Command("helm", "repo", "update", Orkestra)
	update.Stdout = os.Stdout
	update.Stderr = os.Stderr
	_ = update.Run() // non-fatal if offline or repo not yet populated

	// Build Helm args — always use upgrade --install (idempotent)
	args := []string{"upgrade", "--install"}

	args = append(args,
		Orkestra,
		fmt.Sprintf("%s/%s", Orkestra, OrkestraChartName),
		"--namespace", OrkestraNamespace,
		"--create-namespace",
	)

	if version != "" {
		args = append(args, "--version", version)
	}

	for _, f := range valueFiles {
		if f != "" {
			args = append(args, "-f", f)
		}
	}

	// Run Helm
	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing/upgrading Orkestra: %w", err)
	}

	return nil
}
