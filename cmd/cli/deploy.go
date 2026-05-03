//go:build !runtime

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/doktor"
	"github.com/spf13/cobra"
)

var (
	orkSuffix = "-orkestra"
	orkDir    = ".orkestra"
	ork       = "orkestra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Build, push, and deploy the current project to Kubernetes",
	Long: `Build the Docker image, push it, generate the Orkestra bundle, apply it
to the cluster, and patch the CR to trigger a rolling deploy.

  ork deploy --registry ghcr.io/myorg
  ork deploy --registry ghcr.io/myorg --tag v1.2.0
  ork deploy --registry ghcr.io/myorg --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, _ := cmd.Flags().GetString("registry")
		tag, _ := cmd.Flags().GetString("tag")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		noHA, _ := cmd.Flags().GetBool("no-ha")
		noSecure, _ := cmd.Flags().GetBool("no-secure")
		clean, _ := cmd.Flags().GetBool("clean")
		orkestraVersion, _ := cmd.Flags().GetString("orkestra-version")
		values, _ := cmd.Flags().GetString("values")
		upgradeOrkestra, _ := cmd.Flags().GetBool("upgrade-orkestra")

		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		// --- Detect project ---
		info, err := doktor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		// Resolve CR name from .orkestra/app.yaml (written by ork doktor init --name).
		// Fall back to info.AppName only when app.yaml doesn't exist yet.
		appYAML := filepath.Join(dir, orkDir, doktor.ApplicationFile)
		crName, err := doktor.ReadCRName(appYAML)
		if err != nil {
			// app.yaml not yet generated — we'll auto-init below; need --name
			nameFlag, _ := cmd.Flags().GetString("name")
			if nameFlag == "" {
				return fmt.Errorf("run 'ork doktor init --name <app>' first, or pass --name here")
			}
			crName = nameFlag + orkSuffix
		}
		// appName is the bare project name (crName without the -orkestra suffix).
		appName := strings.TrimSuffix(crName, orkSuffix)
		ns := crName + "-ns" // corresponds to "{{ .metadata.name }}-ns"

		if tag == "" {
			tag = info.GitCommit
		}
		if tag == "" {
			tag = "latest"
		}
		if registry == "" {
			return fmt.Errorf("--registry is required (e.g. --registry ghcr.io/myorg)")
		}

		image := doktor.ImageTag(registry, appName, tag)

		// Step 1 — Build
		fmt.Printf("\nBuilding %s...\n", appName)
		fmt.Printf("  → %s\n", image)

		if !dryRun {
			start := time.Now()
			var buildOut bytes.Buffer
			if err := doktor.Build(dir, image, &buildOut); err != nil {
				fmt.Print(buildOut.String())
				return err
			}
			fmt.Printf("  ✓ Built (%ds)\n", int(time.Since(start).Seconds()))

			// Step 2 — Push
			var pushOut bytes.Buffer
			if err := doktor.Push(image, &pushOut); err != nil {
				fmt.Print(pushOut.String())
				return err
			}
			fmt.Println("  ✓ Pushed")
		} else {
			fmt.Println("  ~ dry-run: skipping docker build and push")
		}

		// Step 3 — Generate bundle
		fmt.Println("\nGenerating bundle...")

		initOpts := doktor.GenerateOptions{
			NoHA:     noHA,
			NoSecure: noSecure,
			Clean:    clean,
		}

		bundleDir := filepath.Join(dir, orkDir, "bundle")
		if !dryRun {
			if err := doktor.GenerateBundle(appName, ns, info.Secrets, info.Config, bundleDir); err != nil {
				return err
			}
		}

		if len(info.Secrets) > 0 {
			fmt.Printf("  ✓ %s-secrets  (%d variables from .env)\n", appName, len(info.Secrets))
		}
		if len(info.Config) > 0 {
			fmt.Printf("  ✓ %s-config   (%d variables from .env)\n", appName, len(info.Config))
		}

		// Auto-generate katalog if not present yet.
		katalogPath := filepath.Join(dir, orkDir, "katalog.yaml")
		if !fileExistsAtPath(katalogPath) {
			if !dryRun {
				if err := doktor.Init(info, initOpts); err != nil {
					return fmt.Errorf("generating katalog: %w", err)
				}
			}
			fmt.Println("  ✓ katalog.yaml generated")
		}

		// Generate Orkestra katalog bundle (RBAC, namespace, etc.)
		if fileExistsAtPath(katalogPath) || dryRun {
			if !dryRun {
				genArgs := []string{"generate", "bundle", "-k", katalogPath, "-w", ns, "-o", bundleDir}
				genCmd := exec.Command("ork", genArgs...)
				genCmd.Stdout = os.Stdout
				genCmd.Stderr = os.Stderr
				if err := genCmd.Run(); err != nil {
					return fmt.Errorf("generating bundle: %w", err)
				}
			}
			fmt.Println("  ✓ RBAC + Katalog ConfigMap + namespace")
		}

		// Step 4 — Apply bundle
		fmt.Println("\nApplying to cluster...")

		if !dryRun {
			if !doktor.KubectlAvailable() {
				return fmt.Errorf("kubectl not found in PATH")
			}

			// Apply the bundle.
			bundleFile := filepath.Join(bundleDir, doktor.BundleFile)
			applyBundle := exec.Command("kubectl", "apply", "-f", bundleFile)
			applyBundle.Stdout = os.Stdout
			applyBundle.Stderr = os.Stderr
			if err := applyBundle.Run(); err != nil {
				return fmt.Errorf("applying bundle: %w", err)
			}
			fmt.Println("  ✓ Bundle applied")

			// Now apply app-config + app-secrets
			for _, f := range []string{doktor.AppConfigFile, doktor.AppSecretFile} {
				path := filepath.Join(bundleDir, f)
				if _, err := os.Stat(path); err == nil {
					apply := exec.Command("kubectl", "apply", "-f", path)
					apply.Stdout = os.Stdout
					apply.Stderr = os.Stderr
					if err := apply.Run(); err != nil {
						return fmt.Errorf("applying %s: %w", f, err)
					}
					fmt.Printf("  ✓ %s applied\n", f)
				}
			}

			// Apply the application CR (app.yaml)
			appFile := filepath.Join(dir, orkDir, doktor.ApplicationFile)
			if _, err := os.Stat(appFile); err == nil {
				applyApp := exec.Command("kubectl", "apply", "-f", appFile)
				applyApp.Stdout = os.Stdout
				applyApp.Stderr = os.Stderr
				if err := applyApp.Run(); err != nil {
					return fmt.Errorf("applying application file (%s): %w", doktor.ApplicationFile, err)
				}
				fmt.Printf("  ✓ %s applied\n", doktor.ApplicationFile)
			}

			// Auto-use .orkestra/values.yaml when --values is not explicitly set.
			resolvedValues := values
			if resolvedValues == "" {
				localValues := filepath.Join(dir, orkDir, "values.yaml")
				if fileExistsAtPath(localValues) {
					resolvedValues = localValues
					fmt.Printf("  Using values: %s\n", localValues)
				}
			}

			// Always add the repo (idempotent)
			repoAdd := exec.Command("helm", "repo", "add",
				doktor.Orkestra, doktor.OrkestraChartRepo)
			repoAdd.Stdout = os.Stdout
			repoAdd.Stderr = os.Stderr
			_ = repoAdd.Run() // ignore error: repo may already exist

			// If upgrade flag is set, update the repo
			if upgradeOrkestra {
				updateRepo := exec.Command("helm", "repo", "update", doktor.Orkestra)
				updateRepo.Stdout = os.Stdout
				updateRepo.Stderr = os.Stderr
				if err := updateRepo.Run(); err != nil {
					return fmt.Errorf("updating Orkestra repo: %w", err)
				}
			}

			// Install Orkestra if not present
			if !doktor.OrkestraInstalled() || upgradeOrkestra {
				fmt.Println("  ⠸ Installing Orkestra...")

				if err := doktor.InstallOrUpgradeOrkestra(
					orkestraVersion,
					resolvedValues,
					upgradeOrkestra,
				); err != nil {
					return err
				}

				fmt.Println("  ✓ Orkestra installed")
			} else {
				fmt.Println("  ✓ Orkestra already installed")
			}

			// Step 5 — Verify runtime health before touching workload state.
			fmt.Print("\n  Checking runtime health...")
			health := doktor.CheckRuntimeHealth()
			if !health.Running {
				fmt.Println()
				tail, fetchErr := doktor.FetchRuntimeLogs()
				if fetchErr == nil && tail != "" {
					fmt.Printf("\n--- Runtime log (last 10 lines) ---\n%s\n---\n\n", tail)
				}
				fmt.Printf("  ✗ Orkestra runtime is not healthy: %s\n", health.Reason)
				fmt.Printf("    Full logs:           %s\n", "/tmp/orkestra/runtime.log")
				fmt.Printf("    Control Center logs: %s\n", "/tmp/orkestra/controlcenter.log")
				fmt.Println("    Fix the operator before deploying workloads.")
				return fmt.Errorf("orkestra runtime is not healthy")
			}
			fmt.Println(" ✓")

			// Restart the operator when the Katalog changed so it picks up the
			// new bundle before the workload image is patched.
			if doktor.KatalogChanged(dir) {
				fmt.Println("  Katalog changed — restarting Orkestra runtime")
				if err := doktor.RestartOrkestra(); err != nil {
					return fmt.Errorf("restarting Orkestra: %w", err)
				}
			} else {
				fmt.Println("  Katalog unchanged — Orkestra restart not required")
			}

			// Step 6 — Patch image in CR.
			// Save the current image as an annotation first so 'ork deploy rollback'
			// can restore it instantly without rebuilding anything.
			if prevOut, err := exec.Command("kubectl", "get", "configmap", crName,
				"-n", ns, "-o", `go-template={{index .data "image"}}`).Output(); err == nil {
				if prev := strings.TrimSpace(string(prevOut)); prev != "" && prev != image {
					_ = exec.Command("kubectl", "annotate", "configmap", crName,
						"-n", ns, "orkestra.io/previous-image="+prev, "--overwrite").Run()
				}
			}

			patchCmd := exec.Command("kubectl", "patch", "configmap", crName,
				"-n", ns,
				"--patch", fmt.Sprintf(`{"data":{"image":%q}}`, image),
			)
			if err := patchCmd.Run(); err != nil {
				return fmt.Errorf("patching image in CR: %w", err)
			}
			fmt.Printf("  ✓ Image: %s\n", image)

			// Step 7 — Watch until ready.
			fmt.Println("\nWaiting for deployment...")
			if err := watchUntilReady(crName, ns); err != nil {
				fmt.Printf("  ~ could not confirm readiness: %v\n", err)
			}
		} else {
			fmt.Printf("  ~ dry-run: would apply %s and patch image to %s\n", bundleDir, image)
		}

		return nil
	},
}

var openCmd = &cobra.Command{
	Use:   "open [app-name]",
	Short: "Open the deployed app URL in the browser",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := ""
		if len(args) > 0 {
			appName = args[0]
		} else {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			info, err := doktor.Detect(dir)
			if err != nil {
				return err
			}
			appName = info.AppName
		}

		crName := appName + orkSuffix
		ns := appName + "ns" // corresponds to "{{ .metadata.name }}-ns"
		out, err := exec.Command("kubectl", "get", "configmap", crName,
			"-n", ns, "-o", `jsonpath={.data.url}`).Output()
		if err != nil {
			return fmt.Errorf("reading app URL from CR: %w", err)
		}
		url := strings.TrimSpace(string(out))
		if url == "" {
			return fmt.Errorf("no url in %s CR status — fill in spec.host and redeploy", crName)
		}

		fmt.Printf("Opening %s\n", url)
		return openBrowser(url)
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Roll back to the previous deployed image",
	Long: `Restore the previous image instantly by patching the ConfigMap.
No rebuild or push required — the image was saved as an annotation on deploy.

  ork deploy rollback                       # restore previous image
  ork deploy rollback --image ghcr.io/x:v1  # restore a specific image`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetImage, _ := cmd.Flags().GetString("image")

		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		appYAML := filepath.Join(dir, orkDir, doktor.ApplicationFile)
		crName, err := doktor.ReadCRName(appYAML)
		if err != nil {
			return fmt.Errorf("run 'ork doktor init --name <app>' first: %w", err)
		}
		ns := crName + "-ns"

		if targetImage == "" {
			// Read the previous image from the annotation saved by 'ork deploy'.
			out, err := exec.Command("kubectl", "get", "configmap", crName,
				"-n", ns, "-o", `go-template={{index .metadata.annotations "orkestra.io/previous-image"}}`).Output()
			if err != nil {
				return fmt.Errorf("reading previous-image annotation: %w", err)
			}
			targetImage = strings.TrimSpace(string(out))
			if targetImage == "" || targetImage == "<no value>" {
				return fmt.Errorf("no previous image found — deploy at least once before rolling back")
			}
		}

		// Save the current image so a second rollback call can re-roll-forward.
		if currOut, err := exec.Command("kubectl", "get", "configmap", crName,
			"-n", ns, "-o", `go-template={{index .data "image"}}`).Output(); err == nil {
			if curr := strings.TrimSpace(string(currOut)); curr != "" && curr != targetImage {
				_ = exec.Command("kubectl", "annotate", "configmap", crName,
					"-n", ns, "orkestra.io/previous-image="+curr, "--overwrite").Run()
			}
		}

		fmt.Printf("Rolling back to %s\n", targetImage)
		patch := exec.Command("kubectl", "patch", "configmap", crName,
			"-n", ns, "--patch", fmt.Sprintf(`{"data":{"image":%q}}`, targetImage))
		patch.Stdout = os.Stdout
		patch.Stderr = os.Stderr
		if err := patch.Run(); err != nil {
			return fmt.Errorf("patching image: %w", err)
		}
		fmt.Printf("  ✓ Image set to %s\n", targetImage)

		fmt.Println("\nWaiting for rollback...")
		if err := watchUntilReady(crName, ns); err != nil {
			fmt.Printf("  ~ could not confirm readiness: %v\n", err)
		}
		return nil
	},
}

func init() {
	deployCmd.Flags().StringP("registry", "r", "", "Container registry (e.g. ghcr.io/myorg)")
	deployCmd.Flags().StringP("tag", "t", "", "Image tag (default: git commit SHA)")
	deployCmd.Flags().String("name", "", "App name override (default: read from .orkestra/app.yaml)")
	deployCmd.Flags().Bool("dry-run", false, "Show what would be applied without making changes")
	deployCmd.Flags().Bool("no-ha", false, "Skip HPA and PDB (single replica)")
	deployCmd.Flags().Bool("no-secure", false, "Skip deletion protection and protection labels")
	deployCmd.Flags().Bool("clean", false, "Remove deletion protection webhook on operator shutdown")
	deployCmd.Flags().BoolP("upgrade-orkestra", "u", false, "Upgrade Orkestra to latest version before deployment")
	deployCmd.Flags().String("orkestra-version", "", "Version of Orkestra operator to install")
	deployCmd.Flags().String("values", "", "Path to Helm values.yaml for Orkestra installation")

	rollbackCmd.Flags().String("image", "", "Image to roll back to (default: previous deployed image)")

	deployCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(openCmd)
}

// watchUntilReady polls the CR phase field until it is Ready or until timeout.
func watchUntilReady(appName, ns string) error {
	deadline := time.Now().Add(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C
		out, err := exec.Command("kubectl", "get", "configmap", appName,
			"-n", ns, "-o", `jsonpath={.data.phase}`).Output()
		if err != nil {
			continue
		}
		phase := strings.TrimSpace(string(out))
		switch phase {
		case "Ready":
			// Read URL if available.
			urlOut, _ := exec.Command("kubectl", "get", "configmap", appName,
				"-n", ns, "-o", `jsonpath={.data.url}`).Output()
			url := strings.TrimSpace(string(urlOut))

			commitOut, _ := exec.Command("kubectl", "get", "configmap", appName,
				"-n", ns, "-o", `jsonpath={.data.image}`).Output()
			img := strings.TrimSpace(string(commitOut))

			fmt.Println()
			if url != "" {
				fmt.Printf("  App: %s\n", url)
			}
			fmt.Printf("  Status: Ready\n")
			if img != "" {
				fmt.Printf("  Image: %s\n", img)
			}
			fmt.Println()
			ccOut, _ := exec.Command("kubectl", "get", "configmap", appName,
				"-n", ns, "-o", `go-template={{index .data "controlCenterHost"}}`).Output()
			if ccHost := strings.TrimSpace(string(ccOut)); ccHost != "" {
				fmt.Printf("  Control Center → https://%s\n", ccHost)
			} else {
				fmt.Println("  Control Center → http://orkestra-cc.orkestra-system.svc.cluster.local:8081")
				fmt.Println("                   set controlCenterHost in .orkestra/app.yaml for external access")
			}
			fmt.Printf("  Logs          → kubectl logs -n %s -l ork.io/app=%s -f\n", ns, appName)
			return nil
		case "Deploying", "Pending":
			fmt.Printf("\r  ⠸ %s...", phase)
		}
	}
	return fmt.Errorf("timed out after 5 minutes")
}

func openBrowser(url string) error {
	// Try common openers in order.
	for _, prog := range []string{"xdg-open", "open", "start"} {
		if _, err := exec.LookPath(prog); err == nil {
			return exec.Command(prog, url).Start()
		}
	}
	fmt.Printf("Could not detect browser opener — visit manually: %s\n", url)
	return nil
}

func fileExistsAtPath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
