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
	// Always add repo (idempotent)
	repoAdd := exec.Command("helm", "repo", "add", Orkestra, OrkestraChartRepo)
	repoAdd.Stdout = os.Stdout
	repoAdd.Stderr = os.Stderr
	_ = repoAdd.Run() // ignore "already exists"

	// Update repo only if requested
	if upgrade {
		update := exec.Command("helm", "repo", "update", Orkestra)
		update.Stdout = os.Stdout
		update.Stderr = os.Stderr
		if err := update.Run(); err != nil {
			return fmt.Errorf("updating Orkestra repo: %w", err)
		}
	}

	// Build Helm args
	var args []string
	if upgrade {
		args = []string{"upgrade", "--install"}
	} else {
		args = []string{"install"}
	}

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
