package doctor

import (
	"fmt"
	"os"
	"os/exec"
)

func InstallOrUpgradeOrkestra(version, values string, upgrade bool) error {
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

	if values != "" {
		args = append(args, "-f", values)
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
