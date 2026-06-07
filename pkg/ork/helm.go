package ork

import (
	"fmt"
	"os"
	"os/exec"
)

// InstallOrUpgradeOrkestra installs or upgrades the Orkestra Helm chart
// idempotently. It always runs `helm upgrade --install` so the call is
// safe whether Orkestra is already present or not.
//
// The repo is added and updated before every call to ensure the index is
// current. Version may be empty to use the latest chart. Additional Helm
// flags (e.g. --set, --atomic) are passed through args.
func InstallOrUpgradeOrkestra(version string, valueFiles []string, args ...string) error {
	_ = exec.Command("helm", "repo", "add", Orkestra, OrkestraChartRepo).Run()
	_ = exec.Command("helm", "repo", "update", Orkestra).Run()

	helmArgs := []string{
		"upgrade", "--install",
		Orkestra,
		fmt.Sprintf("%s/%s", Orkestra, OrkestraChartName),
		"--namespace", OrkestraNamespace,
		"--create-namespace",
	}
	if version != "" {
		helmArgs = append(helmArgs, "--version", version)
	}
	for _, f := range valueFiles {
		if f != "" {
			helmArgs = append(helmArgs, "-f", f)
		}
	}
	helmArgs = append(helmArgs, args...)

	cmd := exec.Command("helm", helmArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("installing/upgrading Orkestra: %w\n%s", err, out)
	}
	return nil
}

// BuildControlCenterValues generates a temporary Helm values file that enables
// the Control Center ingress for the given hostname.
// The caller is responsible for removing the file when done.
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
