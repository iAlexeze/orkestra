//go:build !runtime

package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/buildx"
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
		dev, _ := cmd.Flags().GetBool("dev")

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
		appYAML := filepath.Join(dir, orkDir, doktor.ApplicationFile)
		crName, err := doktor.ReadCRName(appYAML)
		if err != nil {
			nameFlag, _ := cmd.Flags().GetString("name")
			if nameFlag == "" {
				return fmt.Errorf("run 'ork doktor init --name <app>' first, or pass --name here")
			}
			crName = nameFlag + orkSuffix
		}
		appName := strings.TrimSuffix(crName, orkSuffix)
		ns := crName + "-ns"

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

		// Load persistent state and global Komposer (both non-fatal if missing).
		state, err := doktor.LoadState()
		if err != nil {
			state = &doktor.DeployState{Projects: make(map[string]*doktor.ProjectState)}
		}
		komposer, _ := doktor.LoadGlobalKomposer()
		komposer.Metadata.Description = "Orkestra Managed Deployment in " + state.ClusterContext

		if !dryRun {
			// Step 0 — Cluster connectivity.
			// --dev spins up a local kind cluster; otherwise verify the current context.
			if dev {
				if !doktor.GoInstalled() {
					return fmt.Errorf("Go is required to install kind — install from https://go.dev/dl/")
				}
				fmt.Printf("\n  Setting up kind cluster '%s'...\n", doktor.KindClusterName)
				if err := doktor.EnsureKindCluster(doktor.KindClusterName); err != nil {
					return fmt.Errorf("setting up kind cluster: %w", err)
				}
			} else if !doktor.ClusterReachable() {
				fmt.Println("\n  Cannot reach Kubernetes cluster.")
				fmt.Println("  Check your kubeconfig, or run with --dev to deploy to a local kind cluster.")
				return fmt.Errorf("cluster not reachable")
			}
		}

		// Load init config (bridges doktor init → deploy).
		initCfg, _ := buildx.LoadInitConfig(dir)

		// ── Multi-app path ────────────────────────────────────────────────────────
		if len(initCfg.Apps) > 0 {
			err := deployMultiApp(deployContext{
				dir:             dir,
				registry:        registry,
				tag:             tag,
				dryRun:          dryRun,
				noHA:            noHA,
				noSecure:        noSecure,
				clean:           clean,
				values:          values,
				orkestraVersion: orkestraVersion,
				upgradeOrkestra: upgradeOrkestra,
				info:            info,
				state:           state,
				komposer:        komposer,
				clusterCtx:      doktor.CurrentContext(),
				initCfg:         initCfg,
			})
			_ = buildx.CleanupInitConfig(dir)
			return err
		}

		// ── Single-app path (legacy) ──────────────────────────────────────────────

		// Show cluster context and currently deployed projects.
		clusterCtx := doktor.CurrentContext()
		deployed := komposer.DeployedProjects()
		fmt.Printf("\nCluster:  %s\n", clusterCtx)
		if len(deployed) > 0 {
			fmt.Printf("Deployed: %s\n", strings.Join(deployed, ", "))
		}

		// Step 1 — Build
		fmt.Printf("\nBuilding %s...\n", appName)
		fmt.Printf("  → %s\n", image)

		if !dryRun {
			start := time.Now()
			var buildOut bytes.Buffer

			composeBuild := buildx.ComposeBuild{
				UseCompose:  initCfg.UseCompose,
				ComposeFile: initCfg.ComposeFile,
			}
			if err := buildx.BuildImage(dir, image, composeBuild, &buildOut); err != nil {
				fmt.Print(buildOut.String())
				return err
			}
			fmt.Printf("  ✓ Built (%ds)\n", int(time.Since(start).Seconds()))

			var pushOut bytes.Buffer
			if err := buildx.PushImage(image, &pushOut); err != nil {
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
		katalogPath := filepath.Join(dir, orkDir, "katalog.yaml")
		absKatalogPath, _ := filepath.Abs(katalogPath)

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
		if !fileExistsAtPath(katalogPath) {
			if !dryRun {
				if err := doktor.Init(info, initOpts); err != nil {
					return fmt.Errorf("generating katalog: %w", err)
				}
			}
			fmt.Println("  ✓ katalog.yaml generated")
		}

		if fileExistsAtPath(katalogPath) || dryRun {
			if !dryRun {
				komposer.RegisterKatalog(absKatalogPath)
				state.ClusterContext = clusterCtx
				komposer.Metadata.Name = clusterCtx
				komposer.Metadata.License = info.License
				if saveErr := komposer.Save(); saveErr != nil {
					return fmt.Errorf("saving komposer: %w", saveErr)
				}

				mergedPath, mergeErr := runKompose()
				if mergeErr != nil {
					return fmt.Errorf("merging katalogs: %w", mergeErr)
				}
				effectiveKatalog := mergedPath
				fmt.Printf("  ✓ Komposer merged (%d projects)\n", len(komposer.DeployedProjects()))

				genArgs := []string{"generate", "bundle", "-k", effectiveKatalog, "-w", ns, "-o", bundleDir}
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
			if err := doktor.EnsureDependencies(); err != nil {
				fmt.Printf("  ~ Could not install dependencies: %v\n", err)
			}

			if info.HasFrontend {
				if err := ensureIngressController(); err != nil {
					fmt.Printf("  ~ Could not install ingress controller: %v\n", err)
					fmt.Println("    Install manually: https://kubernetes.github.io/ingress-nginx/deploy/")
				}
			}

			if err := ensureMetricsServer(); err != nil {
				fmt.Printf("  ~ Could not install metrics server: %v\n", err)
				fmt.Println("    Install manually: https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml")
			}

			bundleFile := filepath.Join(bundleDir, doktor.BundleFile)
			applyBundle := exec.Command("kubectl", "apply", "-f", bundleFile)
			applyBundle.Stdout = os.Stdout
			applyBundle.Stderr = os.Stderr
			if err := applyBundle.Run(); err != nil {
				return fmt.Errorf("applying bundle: %w", err)
			}
			fmt.Println("  ✓ Bundle applied")

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

			if info.HasSMTP || info.HasSlack {
				notifEnvMap := doktor.NotificationEnvVars(info.EnvVars)
				secretYAML := doktor.BuildNotificationSecret(notifEnvMap)
				if secretYAML != "" {
					secretPath := filepath.Join(bundleDir, doktor.NotificationSecretFile)
					if err := os.WriteFile(secretPath, []byte(secretYAML), 0o600); err != nil {
						fmt.Printf("  ~ Could not write notification secret: %v\n", err)
					} else {
						applySecret := exec.Command("kubectl", "apply", "-f", secretPath)
						applySecret.Stdout = os.Stdout
						applySecret.Stderr = os.Stderr
						if err := applySecret.Run(); err != nil {
							fmt.Printf("  ~ Could not apply notification secret: %v\n", err)
						} else {
							fmt.Printf("  ✓ orkestra-notification Secret applied\n")
						}
					}
				}
			}

			resolvedValues := values
			if resolvedValues == "" {
				localValues := filepath.Join(dir, orkDir, "values.yaml")
				if fileExistsAtPath(localValues) {
					resolvedValues = localValues
					fmt.Printf("  Using values: %s\n", localValues)
				}
			}

			repoAdd := exec.Command("helm", "repo", "add",
				doktor.Orkestra, doktor.OrkestraChartRepo)
			repoAdd.Stdout = os.Stdout
			repoAdd.Stderr = os.Stderr
			_ = repoAdd.Run()

			if upgradeOrkestra {
				updateRepo := exec.Command("helm", "repo", "update", doktor.Orkestra)
				updateRepo.Stdout = os.Stdout
				updateRepo.Stderr = os.Stderr
				if err := updateRepo.Run(); err != nil {
					return fmt.Errorf("updating Orkestra repo: %w", err)
				}
			}

			if !doktor.OrkestraInstalled() || upgradeOrkestra {
				fmt.Println("  ⠸ Installing Orkestra...")
				if err := doktor.InstallOrUpgradeOrkestra(
					orkestraVersion, resolvedValues, upgradeOrkestra,
				); err != nil {
					return err
				}
				fmt.Println("  ✓ Orkestra installed")
			} else {
				fmt.Println("  ✓ Orkestra already installed")
			}

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

			if doktor.KatalogChanged(dir) {
				fmt.Println("  Katalog changed — restarting Orkestra runtime")
				if err := doktor.RestartOrkestra(); err != nil {
					return fmt.Errorf("restarting Orkestra: %w", err)
				}
			} else {
				fmt.Println("  Katalog unchanged — Orkestra restart not required")
			}

			state.RecordDeploy(appName, ns, absKatalogPath, image)
			if err := state.Save(); err != nil {
				fmt.Printf("  ~ State save failed: %v\n", err)
			}

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

			fmt.Println("\nWaiting for deployment...")
			if err := watchUntilReady(crName, ns, appName, state); err != nil {
				fmt.Printf("  ~ could not confirm readiness: %v\n", err)
			}
		} else {
			fmt.Printf("  ~ dry-run: would apply %s and patch image to %s\n", bundleDir, image)
		}

		_ = buildx.CleanupInitConfig(dir)
		return nil
	},
}

// deployContext bundles all the parameters needed for both single and multi-app deploy.
type deployContext struct {
	dir             string
	registry        string
	tag             string
	dryRun          bool
	noHA            bool
	noSecure        bool
	clean           bool
	values          string
	orkestraVersion string
	upgradeOrkestra bool
	info            *doktor.ProjectInfo
	state           *doktor.DeployState
	komposer        *doktor.GlobalKomposer
	clusterCtx      string
	initCfg         buildx.InitConfig
}

// deployMultiApp builds, pushes, and deploys each app independently, then
// deploys the combined Komposer so Orkestra manages all apps together.
func deployMultiApp(dc deployContext) error {
	fmt.Printf("\nCluster:  %s\n", dc.clusterCtx)
	if deployed := dc.komposer.DeployedProjects(); len(deployed) > 0 {
		fmt.Printf("Deployed: %s\n", strings.Join(deployed, ", "))
	}

	tag := dc.tag
	if tag == "" {
		tag = dc.info.GitCommit
	}
	if tag == "" {
		tag = "latest"
	}

	type appDeploy struct {
		entry       buildx.AppEntry
		image       string
		crName      string
		ns          string
		appName     string
		bundleDir   string
		katalogPath string
		appInfo     *doktor.ProjectInfo
	}

	// Collect per-app metadata
	var apps []appDeploy
	for _, entry := range dc.initCfg.Apps {
		appName := entry.Name
		image := doktor.ImageTag(dc.registry, appName, tag)
		crName := appName + orkSuffix
		ns := crName + "-ns"
		bundleDir := filepath.Join(dc.dir, orkDir, appName, "bundle")
		katalogPath := filepath.Join(dc.dir, orkDir, appName, "katalog.yaml")

		appInfo, _ := doktor.Detect(entry.Dir)
		if appInfo == nil {
			appInfo = &doktor.ProjectInfo{Dir: entry.Dir, AppName: appName}
		}

		apps = append(apps, appDeploy{
			entry:       entry,
			image:       image,
			crName:      crName,
			ns:          ns,
			appName:     appName,
			bundleDir:   bundleDir,
			katalogPath: katalogPath,
			appInfo:     appInfo,
		})
	}

	// ── Step 1: Build + Push each app ────────────────────────────────────────
	fmt.Printf("\nBuilding %d apps...\n", len(apps))
	if !dc.dryRun {
		for _, app := range apps {
			fmt.Printf("\n  %s → %s\n", app.appName, app.image)
			start := time.Now()
			var buildOut bytes.Buffer

			dockerfile := app.entry.Dockerfile
			composeBuild := buildx.ComposeBuild{Dockerfile: dockerfile}

			appDir := app.entry.Dir
			if appDir == "" {
				appDir = filepath.Join(dc.dir, app.appName)
			}

			if err := buildx.BuildImage(appDir, app.image, composeBuild, &buildOut); err != nil {
				fmt.Print(buildOut.String())
				return fmt.Errorf("building %s: %w", app.appName, err)
			}
			fmt.Printf("  ✓ Built %s (%ds)\n", app.appName, int(time.Since(start).Seconds()))

			var pushOut bytes.Buffer
			if err := buildx.PushImage(app.image, &pushOut); err != nil {
				fmt.Print(pushOut.String())
				return fmt.Errorf("pushing %s: %w", app.appName, err)
			}
			fmt.Printf("  ✓ Pushed %s\n", app.appName)
		}
	} else {
		for _, app := range apps {
			fmt.Printf("  ~ dry-run: would build and push %s → %s\n", app.appName, app.image)
		}
	}

	// ── Step 2: Generate bundles + register katalogs ──────────────────────────
	fmt.Println("\nGenerating bundles...")

	if !dc.dryRun {
		for _, app := range apps {
			if err := doktor.GenerateBundle(app.appName, app.ns, app.appInfo.Secrets, app.appInfo.Config, app.bundleDir); err != nil {
				return fmt.Errorf("generating bundle for %s: %w", app.appName, err)
			}
			if len(app.appInfo.Secrets) > 0 {
				fmt.Printf("  ✓ %s-secrets  (%d variables)\n", app.appName, len(app.appInfo.Secrets))
			}
			if len(app.appInfo.Config) > 0 {
				fmt.Printf("  ✓ %s-config   (%d variables)\n", app.appName, len(app.appInfo.Config))
			}

			absKatalogPath, _ := filepath.Abs(app.katalogPath)
			if fileExistsAtPath(app.katalogPath) {
				dc.komposer.RegisterKatalog(absKatalogPath)
			}
		}

		dc.komposer.Metadata.Name = dc.clusterCtx
		dc.state.ClusterContext = dc.clusterCtx
		if saveErr := dc.komposer.Save(); saveErr != nil {
			return fmt.Errorf("saving komposer: %w", saveErr)
		}

		mergedPath, mergeErr := runKompose()
		if mergeErr != nil {
			return fmt.Errorf("merging katalogs: %w", mergeErr)
		}
		fmt.Printf("  ✓ Komposer merged (%d projects)\n", len(dc.komposer.DeployedProjects()))

		for _, app := range apps {
			genArgs := []string{"generate", "bundle", "-k", mergedPath, "-w", app.ns, "-o", app.bundleDir}
			genCmd := exec.Command("ork", genArgs...)
			genCmd.Stdout = os.Stdout
			genCmd.Stderr = os.Stderr
			if err := genCmd.Run(); err != nil {
				return fmt.Errorf("generating bundle for %s: %w", app.appName, err)
			}
		}
		fmt.Println("  ✓ RBAC + Katalog ConfigMap + namespaces")
	}

	// ── Step 3: Apply bundles ─────────────────────────────────────────────────
	fmt.Println("\nApplying to cluster...")

	if !dc.dryRun {
		if err := doktor.EnsureDependencies(); err != nil {
			fmt.Printf("  ~ Could not install dependencies: %v\n", err)
		}
		if err := ensureMetricsServer(); err != nil {
			fmt.Printf("  ~ Could not install metrics server: %v\n", err)
		}

		for _, app := range apps {
			bundleFile := filepath.Join(app.bundleDir, doktor.BundleFile)
			if fileExistsAtPath(bundleFile) {
				applyBundle := exec.Command("kubectl", "apply", "-f", bundleFile)
				applyBundle.Stdout = os.Stdout
				applyBundle.Stderr = os.Stderr
				if err := applyBundle.Run(); err != nil {
					return fmt.Errorf("applying bundle for %s: %w", app.appName, err)
				}
			}
			for _, f := range []string{doktor.AppConfigFile, doktor.AppSecretFile} {
				path := filepath.Join(app.bundleDir, f)
				if _, err := os.Stat(path); err == nil {
					apply := exec.Command("kubectl", "apply", "-f", path)
					apply.Stdout = os.Stdout
					apply.Stderr = os.Stderr
					if err := apply.Run(); err != nil {
						return fmt.Errorf("applying %s for %s: %w", f, app.appName, err)
					}
				}
			}
			appFile := filepath.Join(dc.dir, orkDir, app.appName, doktor.ApplicationFile)
			if fileExistsAtPath(appFile) {
				applyApp := exec.Command("kubectl", "apply", "-f", appFile)
				applyApp.Stdout = os.Stdout
				applyApp.Stderr = os.Stderr
				if err := applyApp.Run(); err != nil {
					return fmt.Errorf("applying app.yaml for %s: %w", app.appName, err)
				}
			}
		}
		fmt.Println("  ✓ All bundles applied")

		// Install Orkestra once
		resolvedValues := dc.values
		if resolvedValues == "" {
			localValues := filepath.Join(dc.dir, orkDir, "values.yaml")
			if fileExistsAtPath(localValues) {
				resolvedValues = localValues
			}
		}

		repoAdd := exec.Command("helm", "repo", "add", doktor.Orkestra, doktor.OrkestraChartRepo)
		repoAdd.Stdout = os.Stdout
		repoAdd.Stderr = os.Stderr
		_ = repoAdd.Run()

		if !doktor.OrkestraInstalled() || dc.upgradeOrkestra {
			fmt.Println("  ⠸ Installing Orkestra...")
			if err := doktor.InstallOrUpgradeOrkestra(dc.orkestraVersion, resolvedValues, dc.upgradeOrkestra); err != nil {
				return err
			}
			fmt.Println("  ✓ Orkestra installed")
		} else {
			fmt.Println("  ✓ Orkestra already installed")
		}

		// Health check
		fmt.Print("\n  Checking runtime health...")
		health := doktor.CheckRuntimeHealth()
		if !health.Running {
			fmt.Println()
			if tail, fetchErr := doktor.FetchRuntimeLogs(); fetchErr == nil && tail != "" {
				fmt.Printf("\n--- Runtime log (last 10 lines) ---\n%s\n---\n\n", tail)
			}
			fmt.Printf("  ✗ Orkestra runtime is not healthy: %s\n", health.Reason)
			return fmt.Errorf("orkestra runtime is not healthy")
		}
		fmt.Println(" ✓")

		if doktor.KatalogChanged(dc.dir) {
			fmt.Println("  Katalog changed — restarting Orkestra runtime")
			if err := doktor.RestartOrkestra(); err != nil {
				return fmt.Errorf("restarting Orkestra: %w", err)
			}
		}

		// ── Step 4: Patch each CR + watch ────────────────────────────────────
		fmt.Println("\nPatching images...")
		for _, app := range apps {
			absKatalogPath, _ := filepath.Abs(app.katalogPath)
			dc.state.RecordDeploy(app.appName, app.ns, absKatalogPath, app.image)

			if prevOut, err := exec.Command("kubectl", "get", "configmap", app.crName,
				"-n", app.ns, "-o", `go-template={{index .data "image"}}`).Output(); err == nil {
				if prev := strings.TrimSpace(string(prevOut)); prev != "" && prev != app.image {
					_ = exec.Command("kubectl", "annotate", "configmap", app.crName,
						"-n", app.ns, "orkestra.io/previous-image="+prev, "--overwrite").Run()
				}
			}

			patchCmd := exec.Command("kubectl", "patch", "configmap", app.crName,
				"-n", app.ns,
				"--patch", fmt.Sprintf(`{"data":{"image":%q}}`, app.image),
			)
			if err := patchCmd.Run(); err != nil {
				return fmt.Errorf("patching image for %s: %w", app.appName, err)
			}
			fmt.Printf("  ✓ %s → %s\n", app.appName, app.image)
		}
		if err := dc.state.Save(); err != nil {
			fmt.Printf("  ~ State save failed: %v\n", err)
		}

		fmt.Println("\nWaiting for deployments...")
		for _, app := range apps {
			fmt.Printf("  Watching %s...\n", app.appName)
			if err := watchUntilReady(app.crName, app.ns, app.appName, dc.state); err != nil {
				fmt.Printf("  ~ %s: could not confirm readiness: %v\n", app.appName, err)
			}
		}
	} else {
		for _, app := range apps {
			fmt.Printf("  ~ dry-run: would apply bundle and patch %s → %s\n", app.appName, app.image)
		}
	}

	return nil
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
		ns := crName + "-ns"
		out, err := exec.Command("kubectl", "get", "configmap", crName,
			"-n", ns, "-o", `jsonpath={.data.url}`).Output()
		if err != nil {
			return fmt.Errorf("reading app URL from CR: %w", err)
		}
		url := strings.TrimSpace(string(out))
		if url == "" {
			return fmt.Errorf("no url in %s CR status — fill in data.host and redeploy", crName)
		}

		fmt.Printf("Opening %s\n", url)
		return openBrowser(url)
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback [app-name]",
	Short: "Roll back to the previous deployed image",
	Long: `Restore the previous image instantly by patching the ConfigMap.
No rebuild or push required — the image is stored in ~/.orkestra/deploy/state.json.

  ork deploy rollback                       # restore previous image
  ork deploy rollback --image ghcr.io/x:v1  # restore a specific image`,
	Args: cobra.MaximumNArgs(1),
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
		appName := strings.TrimSuffix(crName, orkSuffix)
		if len(args) > 0 {
			appName = args[0]
		}

		if targetImage == "" {
			// Check state.json first, then fall back to the annotation.
			state, _ := doktor.LoadState()
			if state != nil {
				targetImage = state.PreviousImage(appName)
			}
			if targetImage == "" {
				out, annotErr := exec.Command("kubectl", "get", "configmap", crName,
					"-n", ns, "-o", `go-template={{index .metadata.annotations "orkestra.io/previous-image"}}`).Output()
				if annotErr == nil {
					targetImage = strings.TrimSpace(string(out))
					if targetImage == "<no value>" {
						targetImage = ""
					}
				}
			}
			if targetImage == "" {
				return fmt.Errorf("no previous image found for %s — use --image to specify", appName)
			}
		}

		fmt.Printf("Rolling back %s...\n", appName)
		fmt.Printf("  → %s\n\n", targetImage)

		// Swap current ↔ previous in state before patching.
		state, _ := doktor.LoadState()
		if state != nil {
			if p := state.Projects[appName]; p != nil {
				p.PreviousImage = p.CurrentImage
				p.CurrentImage = targetImage
				_ = state.Save()
			}
		}

		// Update annotation so a second rollback can re-roll-forward.
		if currOut, err := exec.Command("kubectl", "get", "configmap", crName,
			"-n", ns, "-o", `go-template={{index .data "image"}}`).Output(); err == nil {
			if curr := strings.TrimSpace(string(currOut)); curr != "" && curr != targetImage {
				_ = exec.Command("kubectl", "annotate", "configmap", crName,
					"-n", ns, "orkestra.io/previous-image="+curr, "--overwrite").Run()
			}
		}

		patch := exec.Command("kubectl", "patch", "configmap", crName,
			"-n", ns, "--patch", fmt.Sprintf(`{"data":{"image":%q}}`, targetImage))
		patch.Stdout = os.Stdout
		patch.Stderr = os.Stderr
		if err := patch.Run(); err != nil {
			return fmt.Errorf("patching image: %w", err)
		}
		fmt.Printf("  ✓ Image set to %s\n", targetImage)

		fmt.Println("\nWaiting for rollback...")
		if err := watchUntilReady(crName, ns, appName, nil); err != nil {
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
	deployCmd.Flags().Bool("dev", false, "Create a local kind cluster (orkestra-playground) for development")

	rollbackCmd.Flags().String("image", "", "Image to roll back to (default: previous deployed image)")

	deployCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(openCmd)
}

// watchUntilReady waits for the Deployment rollout to complete via
// kubectl rollout status. On failure it fetches pod logs and, when a
// previous image is recorded in state, suggests rolling back.
//
// appName is the bare project name (without -orkestra suffix) used for state
// lookup. state may be nil (e.g. when called from rollbackCmd).
func watchUntilReady(crName, ns, appName string, state *doktor.DeployState) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Printf("  ⠸ Waiting for rollout...")

	cmd := exec.CommandContext(ctx,
		"kubectl", "rollout", "status",
		"deployment/"+crName,
		"-n", ns,
		"--timeout=5m",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		fmt.Println()

		// Fetch recent pod logs to help the developer diagnose the failure.
		logOut, logErr := exec.Command("kubectl", "logs",
			"deployment/"+crName, "-n", ns,
			"--tail=20",
		).CombinedOutput()
		if logErr == nil && len(bytes.TrimSpace(logOut)) > 0 {
			fmt.Printf("\n--- %s logs (last 20 lines) ---\n%s\n---\n", crName, string(logOut))
		}

		// Rollback hint — only when this is not the first deploy.
		if state != nil && state.PreviousImage(appName) != "" {
			fmt.Printf("\n  A previous good image is available.\n")
			fmt.Printf("  Roll back with: ork deploy rollback\n")
		}

		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timed out waiting for %s — check: kubectl get pods -n %s", crName, ns)
		}
		return fmt.Errorf("rollout failed for %s — check: kubectl describe deployment %s -n %s",
			crName, crName, ns)
	}

	fmt.Printf("\r  ✓ Deployment ready        \n")
	printReadySummary(crName, ns, state)
	return nil
}

// printReadySummary shows the app URL, image, Control Center link, and a
// checklist of internal service URLs for all deployed projects.
func printReadySummary(crName, ns string, state *doktor.DeployState) {
	get := func(jsonpath string) string {
		out, _ := exec.Command("kubectl", "get", "configmap", crName,
			"-n", ns, "-o", "jsonpath="+jsonpath).Output()
		return strings.TrimSpace(string(out))
	}

	fmt.Println()
	if url := get("{.data.url}"); url != "" {
		fmt.Printf("  App:    %s\n", url)
	}
	fmt.Printf("  Status: Ready\n")
	if img := get("{.data.image}"); img != "" {
		fmt.Printf("  Image:  %s\n", img)
	}
	fmt.Println()

	ccOut, _ := exec.Command("kubectl", "get", "configmap", crName,
		"-n", ns, "-o", `go-template={{index .data "controlCenterHost"}}`).Output()
	if ccHost := strings.TrimSpace(string(ccOut)); ccHost != "" {
		fmt.Printf("  Control Center → https://%s\n", ccHost)
	} else {
		fmt.Println("  Control Center → http://localhost:8081")
		fmt.Printf("                   kubectl port-forward svc/orkestra-cc 8081:8081 -n %s &\n",
			doktor.OrkestraNamespace)
		fmt.Println("                   set controlCenterHost in .orkestra/app.yaml to expose externally")
	}
	fmt.Printf("  Logs          → kubectl logs -n %s -l ork.io/app=%s -f\n", ns, crName)

	// Print internal service URLs for every deployed project so developers can
	// wire them together (e.g. FRONTEND_URL=http://my-frontend-orkestra-svc...).
	if state != nil && len(state.Projects) > 0 {
		fmt.Println()
		fmt.Println("  Internal service URLs:")
		for _, p := range state.Projects {
			svcName := p.Name + "-orkestra-svc"
			portOut, _ := exec.Command("kubectl", "get", "svc", svcName,
				"-n", p.Namespace, "-o", "jsonpath={.spec.ports[0].port}").Output()
			port := strings.TrimSpace(string(portOut))
			if port == "" {
				port = "8080"
			}
			envVar := strings.ToUpper(strings.ReplaceAll(p.Name, "-", "_")) + "_URL"
			internalURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%s", svcName, p.Namespace, port)
			fmt.Printf("    %-30s %s\n", p.Name, internalURL)
			fmt.Printf("    %-30s export %s=%s\n", "", envVar, internalURL)
		}
	}
}

// runKompose runs ork kompose to produce ~/.orkestra/deploy/merged-katalog.yaml
// from all registered Katalogs. Returns the path to the merged file.
func runKompose() (string, error) {
	komposerPath, err := doktor.GlobalKomposerPath()
	if err != nil {
		return "", err
	}
	mergedPath := filepath.Join(filepath.Dir(komposerPath), doktor.RuntimeKatalogPath)
	if err := exec.Command("ork", "kompose", "-k", komposerPath, "-o", mergedPath).Run(); err != nil {
		return "", fmt.Errorf("ork kompose: %w", err)
	}
	return mergedPath, nil
}

// ensureIngressController installs nginx-ingress if no ingress controller is
// found on the current cluster. Uses the kind-specific manifest when the
// current context is a kind cluster; otherwise installs via Helm.
func ensureIngressController() error {
	if doktor.DetectIngressController() != doktor.IngressNone {
		return nil
	}

	fmt.Println("  → Installing ingress controller (nginx)...")

	contextOut, _ := exec.Command("kubectl", "config", "current-context").Output()
	isKind := strings.Contains(string(contextOut), "kind")

	var cmd *exec.Cmd
	if isKind {
		cmd = exec.Command("kubectl", "apply", "-f",
			"https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml")
	} else {
		exec.Command("helm", "repo", "add", "ingress-nginx",
			"https://kubernetes.github.io/ingress-nginx").Run()
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
	fmt.Println("  ✓ Ingress controller ready")
	return nil
}

// ensureMetricsServer installs the Kubernetes metrics-server if it is not
// already present on the cluster. Uses the kind-compatible manifest when the
// current context is a kind cluster; otherwise installs via Helm.
func ensureMetricsServer() error {
	// Detect if metrics-server already exists
	out, _ := exec.Command("kubectl", "get", "deployment", "-n", "kube-system", "metrics-server").CombinedOutput()
	if strings.Contains(string(out), "metrics-server") {
		return nil
	}

	fmt.Println("  → Installing metrics-server...")

	// Detect if current context is kind
	contextOut, _ := exec.Command("kubectl", "config", "current-context").Output()
	isKind := strings.Contains(string(contextOut), "kind")

	// Both paths go through Helm so we can pass --kubelet-insecure-tls only
	// where it is needed. The standard components.yaml does not include that
	// flag and fails on kind because kubelet certs are not CA-signed there.
	exec.Command("helm", "repo", "add", "metrics-server",
		"https://kubernetes-sigs.github.io/metrics-server/").Run()
	exec.Command("helm", "repo", "update").Run()

	args := []string{
		"install", "metrics-server",
		"metrics-server/metrics-server",
		"--namespace", "kube-system",
		"--wait", "--timeout=120s",
	}
	if isKind {
		// kind kubelets use self-signed certs — skip TLS verification
		args = append(args, "--set", "args={--kubelet-insecure-tls}")
	}
	cmd := exec.Command("helm", args...)

	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("  ✓ Metrics server ready")
	return nil
}

func openBrowser(url string) error {
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
