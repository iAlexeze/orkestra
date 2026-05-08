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
	"github.com/orkspace/orkestra/pkg/doctor"
	"github.com/orkspace/orkestra/pkg/tunnel"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
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
		enableMetrics, _ := cmd.Flags().GetBool("enable-metrics")
		expose, _ := cmd.Flags().GetBool("expose")
		tunnelProvider, _ := cmd.Flags().GetString("tunnel-provider")
		tunnelToken, _ := cmd.Flags().GetString("tunnel-token")

		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		// --- Detect project ---
		info, err := doctor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		if registry == "" {
			return fmt.Errorf("--registry is required (e.g. --registry ghcr.io/myorg)")
		}

		// Load persistent state and global Komposer (both non-fatal if missing).
		state, err := doctor.LoadState()
		if err != nil {
			state = &doctor.DeployState{Projects: make(map[string]*doctor.ProjectState)}
		}
		komposer, _ := doctor.LoadGlobalKomposer()
		if state.ClusterContext == "" {
			komposer.Metadata.Description = "Orkestra Managed Deployment in " + state.ClusterContext
		} else {
			komposer.Metadata.Description = "Orkestra Managed Deployment"
		}

		if !dryRun {
			// Step 0 — Cluster connectivity.
			// --dev spins up a local kind cluster; otherwise verify the current context.
			if dev {
				fmt.Printf("\n  Setting up kind cluster '%s'...\n", doctor.KindClusterName)
				if err := doctor.EnsureKindCluster(doctor.KindClusterName); err != nil {
					return fmt.Errorf("setting up kind cluster: %w", err)
				}
			} else if !doctor.ClusterReachable() {
				fmt.Println("\n  Cannot reach Kubernetes cluster.")
				fmt.Println("  Check your kubeconfig, or run with --dev to deploy to a local kind cluster.")
				return fmt.Errorf("cluster not reachable")
			}
		}

		// Load init config (bridges doctor init → deploy).
		initCfg, _ := buildx.LoadInitConfig(dir)

		// ── Multi-app path ────────────────────────────────────────────────────────
		// --use-compose init writes Apps entries but never writes app.yaml, so the
		// crName resolution below is skipped entirely for multi-app projects.
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
				expose:          expose,
				tunnelProvider:  tunnelProvider,
				tunnelToken:     tunnelToken,
				info:            info,
				state:           state,
				komposer:        komposer,
				clusterCtx:      doctor.CurrentContext(),
				initCfg:         initCfg,
			})
			_ = buildx.CleanupInitConfig(dir)
			return err
		}

		// ── Single-app path (legacy) ──────────────────────────────────────────────

		// Resolve CR name from .orkestra/app.yaml (written by ork doctor init --name).
		appYAML := filepath.Join(dir, orkDir, doctor.ApplicationFile)
		crName, err := doctor.ReadCRName(appYAML)
		if err != nil {
			nameFlag, _ := cmd.Flags().GetString("name")
			if nameFlag == "" {
				return fmt.Errorf("run 'ork doctor init --name <app>' first, or pass --name here")
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

		image := doctor.ImageTag(registry, appName, tag)

		// Show cluster context and currently deployed projects.
		clusterCtx := doctor.CurrentContext()
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
			fmt.Printf("  %s Built (%ds)\n", utils.SuccessMark(), int(time.Since(start).Seconds()))

			var pushOut bytes.Buffer
			if err := buildx.PushImage(image, &pushOut); err != nil {
				fmt.Print(pushOut.String())
				return err
			}
			fmt.Printf("  %s Pushed", utils.SuccessMark())
		} else {
			fmt.Println("  ~ dry-run: skipping docker build and push")
		}

		// Step 3 — Generate bundle
		fmt.Println("\nGenerating bundle...")

		initOpts := doctor.GenerateOptions{
			NoHA:     noHA,
			NoSecure: noSecure,
			Clean:    clean,
		}

		bundleDir := filepath.Join(dir, orkDir, "bundle")
		katalogPath := filepath.Join(dir, orkDir, "katalog.yaml")
		absKatalogPath, _ := filepath.Abs(katalogPath)

		if !dryRun {
			if err := doctor.GenerateBundle(appName, ns, info.Secrets, info.Config, bundleDir); err != nil {
				return err
			}
		}

		if len(info.Secrets) > 0 {
			fmt.Printf("  %s %s-secrets  (%d variables from .env)\n", utils.SuccessMark(), appName, len(info.Secrets))
		}
		if len(info.Config) > 0 {
			fmt.Printf("  %s %s-config   (%d variables from .env)\n", utils.SuccessMark(), appName, len(info.Config))
		}

		// Auto-generate katalog if not present yet.
		if !fileExistsAtPath(katalogPath) {
			if !dryRun {
				if err := doctor.Init(info, initOpts); err != nil {
					return fmt.Errorf("generating katalog: %w", err)
				}
			}
			fmt.Printf("  %s katalog.yaml generated", utils.SuccessMark())
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
				if err := doctor.DeduplicateKatalogGVKs(mergedPath); err != nil {
					return fmt.Errorf("deduplicating katalog GVKs: %w", err)
				}
				effectiveKatalog := mergedPath
				fmt.Printf("  %s Komposer merged (%d projects)\n", utils.SuccessMark(), len(komposer.DeployedProjects()))

				genArgs := []string{"generate", "bundle", "-f", effectiveKatalog, "-w", ns, "-o", bundleDir}
				genCmd := exec.Command("ork", genArgs...)
				genCmd.Stdout = os.Stdout
				genCmd.Stderr = os.Stderr
				if err := genCmd.Run(); err != nil {
					return fmt.Errorf("generating bundle: %w", err)
				}
			}
			fmt.Printf("  %s RBAC + Katalog ConfigMap + namespace", utils.SuccessMark())
		}

		// Step 4 — Apply bundle
		fmt.Println("\nApplying to cluster...")

		if !dryRun {
			if err := doctor.EnsureDependencies(); err != nil {
				fmt.Printf("  ~ Could not install dependencies: %v\n", err)
			}

			if err := doctor.EnsureIngressController(); err != nil {
				fmt.Printf("  ~ Could not install ingress controller: %v\n", err)
				fmt.Println("    Install manually: https://kubernetes.github.io/ingress-nginx/deploy/")
			}

			if enableMetrics {
				if err := doctor.EnsureMetricsServer(); err != nil {
					fmt.Printf("  ~ Could not install metrics server: %v\n", err)
					fmt.Println("    Install manually: https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml")
				}
			}

			bundleFile := filepath.Join(bundleDir, doctor.BundleFile)
			applyBundle := exec.Command("kubectl", "apply", "-f", bundleFile)
			applyBundle.Stdout = os.Stdout
			applyBundle.Stderr = os.Stderr
			if err := applyBundle.Run(); err != nil {
				return fmt.Errorf("applying bundle: %w", err)
			}
			fmt.Printf("  %s Bundle applied", utils.SuccessMark())

			for _, f := range []string{doctor.AppConfigFile, doctor.AppSecretFile} {
				path := filepath.Join(bundleDir, f)
				if _, err := os.Stat(path); err == nil {
					apply := exec.Command("kubectl", "apply", "-f", path)
					apply.Stdout = os.Stdout
					apply.Stderr = os.Stderr
					if err := apply.Run(); err != nil {
						return fmt.Errorf("applying %s: %w", f, err)
					}
					fmt.Printf("  %s %s applied\n", utils.SuccessMark(), f)
				}
			}

			appFile := filepath.Join(dir, orkDir, doctor.ApplicationFile)
			if _, err := os.Stat(appFile); err == nil {
				applyApp := exec.Command("kubectl", "apply", "-f", appFile)
				applyApp.Stdout = os.Stdout
				applyApp.Stderr = os.Stderr
				if err := applyApp.Run(); err != nil {
					return fmt.Errorf("applying application file (%s): %w", doctor.ApplicationFile, err)
				}
				fmt.Printf("  %s %s applied\n", utils.SuccessMark(), doctor.ApplicationFile)
			}

			if info.HasSMTP || info.HasSlack {
				notifEnvMap := doctor.NotificationEnvVars(info.EnvVars)
				secretYAML := doctor.BuildNotificationSecret(notifEnvMap)
				if secretYAML != "" {
					secretPath := filepath.Join(bundleDir, doctor.NotificationSecretFile)
					if err := os.WriteFile(secretPath, []byte(secretYAML), 0o600); err != nil {
						fmt.Printf("  ~ Could not write notification secret: %v\n", err)
					} else {
						applySecret := exec.Command("kubectl", "apply", "-f", secretPath)
						applySecret.Stdout = os.Stdout
						applySecret.Stderr = os.Stderr
						if err := applySecret.Run(); err != nil {
							fmt.Printf("  ~ Could not apply notification secret: %v\n", err)
						} else {
							fmt.Printf("  %s orkestra-notification Secret applied\n", utils.SuccessMark())
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
				doctor.Orkestra, doctor.OrkestraChartRepo)
			repoAdd.Stdout = os.Stdout
			repoAdd.Stderr = os.Stderr
			_ = repoAdd.Run()

			if upgradeOrkestra {
				updateRepo := exec.Command("helm", "repo", "update", doctor.Orkestra)
				updateRepo.Stdout = os.Stdout
				updateRepo.Stderr = os.Stderr
				if err := updateRepo.Run(); err != nil {
					return fmt.Errorf("updating Orkestra repo: %w", err)
				}
			}

			if !doctor.OrkestraInstalled() || upgradeOrkestra {
				fmt.Println("  ⠸ Installing Orkestra...")
				if err := doctor.InstallOrUpgradeOrkestra(
					orkestraVersion, resolvedValues, upgradeOrkestra,
				); err != nil {
					return err
				}
				fmt.Printf("  %s Orkestra installed", utils.SuccessMark())
			} else {
				fmt.Printf("  %s Orkestra already installed", utils.SuccessMark())
			}

			fmt.Print("\n  Checking runtime health...")
			health := doctor.CheckRuntimeHealth()
			if !health.Running {
				fmt.Println()
				tail, fetchErr := doctor.FetchRuntimeLogs()
				if fetchErr == nil && tail != "" {
					fmt.Printf("\n--- Runtime log (last 10 lines) ---\n%s\n---\n\n", tail)
				}
				fmt.Printf("  %s Orkestra runtime is not healthy: %s\n", utils.FailureMark(), health.Reason)
				fmt.Printf("    Full logs:           %s\n", "/tmp/orkestra/runtime.log")
				fmt.Printf("    Control Center logs: %s\n", "/tmp/orkestra/controlcenter.log")
				fmt.Println("    Fix the operator before deploying workloads.")
				return fmt.Errorf("orkestra runtime is not healthy")
			}
			fmt.Printf(" %s", utils.SuccessMark())

			if doctor.KatalogChanged(dir) {
				fmt.Println("  Katalog changed — restarting Orkestra runtime")
				if err := doctor.RestartOrkestra(); err != nil {
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
			fmt.Printf("  %s Image: %s\n", utils.SuccessMark(), image)

			fmt.Println("\nWaiting for deployment...")
			if err := watchUntilReady(crName, ns, appName, state); err != nil {
				fmt.Printf("  ~ could not confirm readiness: %v\n", err)
			}

			if expose {
				exposeApp(cmd.Context(), appName, ns, tunnelProvider, tunnelToken)
				exposeControlCenter(cmd.Context(), tunnelProvider, tunnelToken)
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
	expose          bool
	tunnelProvider  string
	tunnelToken     string
	info            *orktypes.ProjectInfo
	state           *doctor.DeployState
	komposer        *doctor.GlobalKomposer
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
		appInfo     *orktypes.ProjectInfo
	}

	// Collect per-app metadata
	var apps []appDeploy
	for _, entry := range dc.initCfg.Apps {
		appName := entry.Name
		image := doctor.ImageTag(dc.registry, appName, tag)
		crName := appName + orkSuffix
		ns := crName + "-ns"
		bundleDir := filepath.Join(dc.dir, orkDir, appName, "bundle")
		katalogPath := filepath.Join(dc.dir, orkDir, appName, "katalog.yaml")

		appInfo, _ := doctor.Detect(entry.Dir)
		if appInfo == nil {
			appInfo = &orktypes.ProjectInfo{Dir: entry.Dir, AppName: appName}
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
			fmt.Printf("  %s Built %s (%ds)\n", utils.SuccessMark(), app.appName, int(time.Since(start).Seconds()))

			var pushOut bytes.Buffer
			if err := buildx.PushImage(app.image, &pushOut); err != nil {
				fmt.Print(pushOut.String())
				return fmt.Errorf("pushing %s: %w", app.appName, err)
			}
			fmt.Printf("  %s Pushed %s\n", utils.SuccessMark(), app.appName)
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
			if err := doctor.GenerateBundle(app.appName, app.ns, app.appInfo.Secrets, app.appInfo.Config, app.bundleDir); err != nil {
				return fmt.Errorf("generating bundle for %s: %w", app.appName, err)
			}
			if len(app.appInfo.Secrets) > 0 {
				fmt.Printf("  %s %s-secrets  (%d variables)\n", utils.SuccessMark(), app.appName, len(app.appInfo.Secrets))
			}
			if len(app.appInfo.Config) > 0 {
				fmt.Printf("  %s %s-config   (%d variables)\n", utils.SuccessMark(), app.appName, len(app.appInfo.Config))
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
		if err := doctor.DeduplicateKatalogGVKs(mergedPath); err != nil {
			return fmt.Errorf("deduplicating katalog GVKs: %w", err)
		}
		fmt.Printf("  %s Komposer merged (%d projects)\n", utils.SuccessMark(), len(dc.komposer.DeployedProjects()))

		for _, app := range apps {
			genArgs := []string{"generate", "bundle", "-f", mergedPath, "-w", app.ns, "-o", app.bundleDir}
			genCmd := exec.Command("ork", genArgs...)
			genCmd.Stdout = os.Stdout
			genCmd.Stderr = os.Stderr
			if err := genCmd.Run(); err != nil {
				return fmt.Errorf("generating bundle for %s: %w", app.appName, err)
			}
		}
		fmt.Printf("  %s RBAC + Katalog ConfigMap + namespaces", utils.SuccessMark())
	}

	// ── Step 3: Apply bundles ─────────────────────────────────────────────────
	fmt.Println("\nApplying to cluster...")

	if !dc.dryRun {
		if err := doctor.EnsureDependencies(); err != nil {
			fmt.Printf("  ~ Could not install dependencies: %v\n", err)
		}
		if err := doctor.EnsureMetricsServer(); err != nil {
			fmt.Printf("  ~ Could not install metrics server: %v\n", err)
		}

		for _, app := range apps {
			bundleFile := filepath.Join(app.bundleDir, doctor.BundleFile)
			if fileExistsAtPath(bundleFile) {
				applyBundle := exec.Command("kubectl", "apply", "-f", bundleFile)
				applyBundle.Stdout = os.Stdout
				applyBundle.Stderr = os.Stderr
				if err := applyBundle.Run(); err != nil {
					return fmt.Errorf("applying bundle for %s: %w", app.appName, err)
				}
			}
			for _, f := range []string{doctor.AppConfigFile, doctor.AppSecretFile} {
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
			appFile := filepath.Join(dc.dir, orkDir, app.appName, doctor.ApplicationFile)
			if fileExistsAtPath(appFile) {
				applyApp := exec.Command("kubectl", "apply", "-f", appFile)
				applyApp.Stdout = os.Stdout
				applyApp.Stderr = os.Stderr
				if err := applyApp.Run(); err != nil {
					return fmt.Errorf("applying app.yaml for %s: %w", app.appName, err)
				}
			}
		}
		fmt.Printf("  %s All bundles applied", utils.SuccessMark())

		// Install Orkestra once
		resolvedValues := dc.values
		if resolvedValues == "" {
			localValues := filepath.Join(dc.dir, orkDir, "values.yaml")
			if fileExistsAtPath(localValues) {
				resolvedValues = localValues
			}
		}

		repoAdd := exec.Command("helm", "repo", "add", doctor.Orkestra, doctor.OrkestraChartRepo)
		repoAdd.Stdout = os.Stdout
		repoAdd.Stderr = os.Stderr
		_ = repoAdd.Run()

		if !doctor.OrkestraInstalled() || dc.upgradeOrkestra {
			fmt.Println("  ⠸ Installing Orkestra...")
			if err := doctor.InstallOrUpgradeOrkestra(dc.orkestraVersion, resolvedValues, dc.upgradeOrkestra); err != nil {
				return err
			}
			fmt.Printf("  %s Orkestra installed", utils.SuccessMark())
		} else {
			fmt.Printf("  %s Orkestra already installed", utils.SuccessMark())
		}

		// Health check
		fmt.Print("\n  Checking runtime health...")
		health := doctor.CheckRuntimeHealth()
		if !health.Running {
			fmt.Println()
			if tail, fetchErr := doctor.FetchRuntimeLogs(); fetchErr == nil && tail != "" {
				fmt.Printf("\n--- Runtime log (last 10 lines) ---\n%s\n---\n\n", tail)
			}
			fmt.Printf("  %s Orkestra runtime is not healthy: %s\n", utils.FailureMark(), health.Reason)
			return fmt.Errorf("orkestra runtime is not healthy")
		}
		fmt.Printf(" %s", utils.SuccessMark())

		if doctor.KatalogChanged(dc.dir) {
			fmt.Println("  Katalog changed — restarting Orkestra runtime")
			if err := doctor.RestartOrkestra(); err != nil {
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
			fmt.Printf("  %s %s → %s\n", utils.SuccessMark(), app.appName, app.image)
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

		if dc.expose {
			for _, app := range apps {
				exposeApp(context.Background(), app.appName, app.ns, dc.tunnelProvider, dc.tunnelToken)
			}
			exposeControlCenter(context.Background(), dc.tunnelProvider, dc.tunnelToken)
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
			info, err := doctor.Detect(dir)
			if err != nil {
				return err
			}
			appName = info.Name
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

		appYAML := filepath.Join(dir, orkDir, doctor.ApplicationFile)
		crName, err := doctor.ReadCRName(appYAML)
		if err != nil {
			return fmt.Errorf("run 'ork doctor init --name <app>' first: %w", err)
		}
		ns := crName + "-ns"
		appName := strings.TrimSuffix(crName, orkSuffix)
		if len(args) > 0 {
			appName = args[0]
		}

		if targetImage == "" {
			// Check state.json first, then fall back to the annotation.
			state, _ := doctor.LoadState()
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
		state, _ := doctor.LoadState()
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
		fmt.Printf("  %s Image set to %s\n", utils.SuccessMark(), targetImage)

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
	deployCmd.Flags().Bool("enable-metrics", false, "Install metrics server to the cluster")
	deployCmd.Flags().Bool("expose", false, "Expose the deployed app via a public HTTPS tunnel (cloudflared or ngrok)")
	deployCmd.Flags().String("tunnel-provider", "", "Tunnel provider: cloudflared (default) or ngrok")
	deployCmd.Flags().String("tunnel-token", "", "Auth token for ngrok tunnels")

	rollbackCmd.Flags().String("image", "", "Image to roll back to (default: previous deployed image)")

	deployCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(openCmd)

	// Shadow global flags so they don't appear under `ork init`
	deployCmd.Flags().Bool("debug", false, "")
	deployCmd.Flags().String("kubeconfig", "", "")
	deployCmd.Flags().StringSlice("katalog", nil, "")
	deployCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	deployCmd.Flags().MarkHidden("debug")
	deployCmd.Flags().MarkHidden("kubeconfig")
	deployCmd.Flags().MarkHidden("katalog")
	deployCmd.Flags().MarkHidden("verbose")
}

// watchUntilReady waits for the Deployment rollout to complete via
// kubectl rollout status. On failure it fetches pod logs and, when a
// previous image is recorded in state, suggests rolling back.
//
// appName is the bare project name (without -orkestra suffix) used for state
// lookup. state may be nil (e.g. when called from rollbackCmd).
func watchUntilReady(crName, ns, appName string, state *doctor.DeployState) error {
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

	fmt.Printf("\r  %s Deployment ready        \n", utils.SuccessMark())
	printReadySummary(crName, ns, state)
	return nil
}

// printReadySummary shows the app URL, image, Control Center link, and a
// checklist of internal service URLs for all deployed projects.
func printReadySummary(crName, ns string, state *doctor.DeployState) {
	get := func(jsonpath string) string {
		out, _ := exec.Command("kubectl", "get", "configmap", crName,
			"-n", ns, "-o", "jsonpath="+jsonpath).Output()
		return strings.TrimSpace(string(out))
	}

	// Load live tunnel URLs (best-effort — nil map is safe to read).
	tunnels, _ := tunnel.LoadAllStates()
	appName := strings.TrimSuffix(crName, orkSuffix)

	fmt.Println()
	// App URL: prefer tunnel URL, fall back to ingress URL.
	if ts, ok := tunnels[appName]; ok && ts.IsAlive() {
		fmt.Printf("  App:    %s\n", ts.URL)
	} else if url := get("{.data.url}"); url != "" {
		fmt.Printf("  App:    %s\n", url)
	}
	fmt.Printf("  Status: Ready\n")
	if img := get("{.data.image}"); img != "" {
		fmt.Printf("  Image:  %s\n", img)
	}
	fmt.Println()

	// Control Center URL: prefer tunnel URL, then static host, then port-forward hint.
	if ts, ok := tunnels["controlcenter"]; ok && ts.IsAlive() {
		fmt.Printf("  Control Center → %s\n", ts.URL)
	} else {
		ccOut, _ := exec.Command("kubectl", "get", "configmap", crName,
			"-n", ns, "-o", `go-template={{index .data "controlCenterHost"}}`).Output()
		if ccHost := strings.TrimSpace(string(ccOut)); ccHost != "" {
			fmt.Printf("  Control Center → https://%s\n", ccHost)
		} else {
			fmt.Println("  Control Center → http://localhost:8081")
			fmt.Printf("                   kubectl port-forward svc/orkestra-cc 8081:8081 -n %s &\n",
				doctor.OrkestraNamespace)
			fmt.Println("                   set controlCenterHost in .orkestra/app.yaml to expose externally")
		}
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
	komposerPath, err := doctor.GlobalKomposerPath()
	if err != nil {
		return "", err
	}
	mergedPath := filepath.Join(filepath.Dir(komposerPath), doctor.RuntimeKatalogPath)
	if err := exec.Command("ork", "kompose", "-f", komposerPath, "-o", mergedPath).Run(); err != nil {
		return "", fmt.Errorf("ork kompose: %w", err)
	}
	return mergedPath, nil
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

// exposeApp starts a tunnel for the given app and prints the public URL.
// It port-forwards to the app's K8s service (<appName>-orkestra-svc) so the
// tunnel survives after the deploy command exits regardless of ingress setup.
func exposeApp(ctx context.Context, appName, ns, provider, token string) {
	fmt.Printf("\n  Starting tunnel for %s...\n", appName)
	url, err := tunnel.Expose(ctx, tunnel.ExposeOptions{
		Name:        appName,
		Provider:    provider,
		Token:       token,
		ServiceName: appName + orkSuffix + "-svc",
		ServicePort: "8080",
		Namespace:   ns,
	})
	if err != nil {
		fmt.Printf("  ~ tunnel: %v\n", err)
		return
	}
	fmt.Printf("  %s App: %s\n", utils.SuccessMark(), url)
}

// exposeControlCenter starts a tunnel for the Orkestra Control Center.
func exposeControlCenter(ctx context.Context, provider, token string) {
	fmt.Println("\n  Starting tunnel for controlcenter...")
	url, err := tunnel.Expose(ctx, tunnel.ExposeOptions{
		Name:        "controlcenter",
		Provider:    provider,
		Token:       token,
		ServiceName: doctor.OrkestraControlCenter,
		Namespace:   doctor.OrkestraNamespace,
		ServicePort: doctor.OrkestraControlCenterPort,
		PortForward: true,
	})
	if err != nil {
		fmt.Printf("  ~ tunnel (controlcenter): %v\n", err)
		return
	}
	fmt.Printf("  %s Control Center: %s\n", utils.SuccessMark(), url)
}
