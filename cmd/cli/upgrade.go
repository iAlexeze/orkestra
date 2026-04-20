package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/orkspace/orkestra/pkg/version"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the Orkestra CLI to the latest or a specific version",
	Long: `Upgrade the Orkestra CLI (and optionally the Control Center) to the latest release.

Examples:
  ork upgrade
  ork upgrade --version v1.4.0
  ork upgrade -v v1.4.0
  ork upgrade --runtime-only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		requestedVersion, _ := cmd.Flags().GetString("version")
		runtimeOnly, _ := cmd.Flags().GetBool("runtime-only")

		if check {
			return runUpgradeCheck(requestedVersion)
		}

		return runUpgrade(requestedVersion, runtimeOnly)
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.Flags().Bool("runtime-only", false, "Upgrade only the ork runtime (skip orkcc)")
	upgradeCmd.Flags().Bool("check", false, "Check if a newer Orkestra version is available")

	// Shadow global flags so they don't appear under `ork upgrade`
	upgradeCmd.Flags().Bool("debug", false, "")
	upgradeCmd.Flags().String("kubeconfig", "", "")
	upgradeCmd.Flags().StringSlice("katalog", nil, "")
	upgradeCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	upgradeCmd.Flags().MarkHidden("debug")
	upgradeCmd.Flags().MarkHidden("kubeconfig")
	upgradeCmd.Flags().MarkHidden("katalog")
	upgradeCmd.Flags().MarkHidden("verbose")
}

//
// ──────────────────────────────────────────────────────────────────────────────
//  Upgrade Logic
// ──────────────────────────────────────────────────────────────────────────────
//

func runUpgrade(requestedVersion string, runtimeOnly bool) error {
	printBanner()

	fmt.Printf("Upgrading Orkestra CLI...\n")

	// Detect platform (linux_amd64, darwin_arm64, etc.)
	platform, err := detectPlatform()
	if err != nil {
		return err
	}

	// Resolve version (latest or user-specified)
	version, err := resolveVersion(requestedVersion)
	if err != nil {
		return err
	}

	fmt.Printf("→ Target version: %s\n", version)
	fmt.Printf("→ Platform: %s\n\n", platform)

	// Install ork runtime
	if err := installBinary("ork", platform, version); err != nil {
		return err
	}

	// Install orkcc unless runtime-only
	if !runtimeOnly {
		_ = installBinary("orkcc", platform, version)
	}

	fmt.Println()
	fmt.Println("✔ Upgrade complete")
	fmt.Println()

	// Print version
	fmt.Println("Current version:")
	fmt.Println("  ork version")
	fmt.Println()

	return nil
}

//
// ──────────────────────────────────────────────────────────────────────────────
//  Platform Detection
// ──────────────────────────────────────────────────────────────────────────────
//

func detectPlatform() (string, error) {
	var osName, arch string

	switch runtime.GOOS {
	case "linux":
		osName = "linux"
	case "darwin":
		osName = "darwin"
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	return fmt.Sprintf("%s_%s", osName, arch), nil
}

//
// ──────────────────────────────────────────────────────────────────────────────
//  Version Resolution (latest or pinned)
// ──────────────────────────────────────────────────────────────────────────────
//

func resolveVersion(requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}

	// Fetch latest release tag from GitHub API
	resp, err := http.Get("https://api.github.com/repos/orkspace/orkestra/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()

	type ghRelease struct {
		Tag string `json:"tag_name"`
	}

	var data struct {
		TagName string `json:"tag_name"`
	}

	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &data)

	if data.TagName == "" {
		return "", fmt.Errorf("could not resolve latest version")
	}

	return data.TagName, nil
}

// should find a way to use requested
// Not a concern for now
func runUpgradeCheck(requested string) error {
	printBanner()

	fmt.Printf("Checking for Orkestra CLI updates...\n")

	// Current version from ldflags
	current := version.Version

	// Resolve latest version
	latest, err := resolveVersion("") // "" fetches the latest
	if err != nil {
		return err
	}

	fmt.Printf("Current version: %s\n", current)
	fmt.Printf("Latest version:  %s\n\n", latest)

	if current == latest {
		fmt.Println("✔ You are already running the latest version.")
		return nil
	}

	fmt.Println("⚠ A new version is available!")
	fmt.Println("Run:")
	fmt.Println("  ork upgrade")
	fmt.Println()

	return nil
}

//
// ──────────────────────────────────────────────────────────────────────────────
//  Install a Single Binary (ork or orkcc)
// ──────────────────────────────────────────────────────────────────────────────
//

func installBinary(binary, platform, version string) error {
	archive := fmt.Sprintf("%s_%s.tar.gz", binary, platform)
	url := fmt.Sprintf("https://github.com/orkspace/orkestra/releases/download/%s/%s", version, archive)

	fmt.Printf("→ Downloading %s...\n", archive)

	tmp, err := os.CreateTemp("", archive)
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("  ⚠ %s not available for this version — skipping\n", binary)
		return nil
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}

	fmt.Printf("→ Extracting %s...\n", archive)

	if err := extractTarGz(tmp.Name(), binary); err != nil {
		return err
	}

	installPath := "/usr/local/bin/" + binary

	fmt.Printf("→ Installing %s to %s...\n", binary, installPath)

	if err := os.Chmod(binary, 0755); err != nil {
		return err
	}

	// Move binary into place (sudo if needed)
	if err := os.Rename(binary, installPath); err != nil {
		return fmt.Errorf("failed to install %s: %w", binary, err)
	}

	fmt.Printf("✔ Installed %s %s\n", binary, version)
	return nil
}

//
// ──────────────────────────────────────────────────────────────────────────────
//  Extract tar.gz containing a single binary
// ──────────────────────────────────────────────────────────────────────────────
//

func extractTarGz(path, binary string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg && header.Name == binary {
			out, err := os.Create(binary)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			return nil
		}
	}

	return fmt.Errorf("binary %s not found in archive", binary)
}
