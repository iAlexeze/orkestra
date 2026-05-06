//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/buildx"
	"github.com/orkspace/orkestra/pkg/doctor"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Examine the project and show what Orkestra found",
	Long: `Examine the current directory and report what Orkestra discovered.

  ork doctor                       show project analysis
  ork doctor --app app,frontend    show per-app analysis (multi-app project)
  ork doctor init                  generate .orkestra/ katalog and app config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		appFlag, _ := cmd.Flags().GetString("app")
		noHA, _ := cmd.Flags().GetBool("no-ha")
		noSecure, _ := cmd.Flags().GetBool("no-secure")

		// Multi-app scan — either from --app flag or from compose buildable services
		if appFlag != "" {
			return doctorScanMultiApp(dir, strings.Split(appFlag, ","), noHA, noSecure)
		}

		// Single-app scan (default)
		fmt.Println("\nExamining project...")
		fmt.Println()

		info, err := doctor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		// Missing Dockerfile + no compose build context → show language-aware error
		if !info.HasDockerfile && !info.HasCompose {
			if info.Language != orktypes.LangUnknown {
				tmpl := doctor.DockerfileTemplate(info.Language)
				if tmpl != "" {
					return fmt.Errorf(`
──────────────────────────────────────────────
❌ Cannot build application "%s"

No Dockerfile or Compose build context was found.

Detected language: %s  (%s)

Provide at least one of:
  • Dockerfile
  • Containerfile
  • docker-compose build context

Suggested starter Dockerfile:
------------------------------------------------------------
%s
------------------------------------------------------------

Docs: https://orkestra.dev/docs/build
──────────────────────────────────────────────`, info.Name, info.Language, info.LangMarker, tmpl)
				}
			}

			// Unknown language fallback
			return fmt.Errorf(`
──────────────────────────────────────────────
❌ Cannot build application "%s"

No Dockerfile or Compose build context was found.

Orkestra could not detect the language of this project.

Please add a Dockerfile to the project root.
──────────────────────────────────────────────`, info.Name)
		}

		var dockerfilePath string
		if info.HasDockerfile {
			dockerfilePath = filepath.Base(info.DockerfilePath)
			fmt.Printf("  ✓ %s found\n", dockerfilePath)
		} else {
			fmt.Println("  ✗ No Dockerfile or Containerfile found — add one to build an image")
		}

		if info.GitCommit != "" {
			fmt.Printf("  ✓ Git repository — commit: %s\n", info.GitCommit)
		} else {
			fmt.Println("  ✗ Not a git repository — run 'git init'")
		}

		if info.Language != orktypes.LangUnknown {
			fmt.Printf("  ✓ Language: %s  (%s)\n", info.Language, info.LangMarker)
		} else {
			fmt.Println("  ~ Language: unknown")
		}

		fmt.Printf("  ✓ Port: %s\n", info.Port)

		if len(info.EnvVars) > 0 {
			fmt.Printf("  ✓ .env — %d variables\n", len(info.EnvVars))
			fmt.Printf("      %d config  (# ork:cfg)\n", len(info.Config))
			fmt.Printf("      %d secrets (default)\n", len(info.Secrets))
		} else {
			fmt.Println("  ~ .env not found — variables can be added later")
		}

		// docker-compose detection
		composeFile := filepath.Base(info.ComposePath)
		if info.HasCompose {
			fmt.Printf("  ✓ (%s)\n", composeFile)
			cf, cfErr := doctor.ParseCompose(info.ComposePath)
			if cfErr == nil {
				buildable, stateful := doctor.ClassifyServices(cf)
				if len(stateful) > 0 {
					fmt.Println()
					fmt.Printf("💡 Infrastructure services detected in %s:\n", composeFile)
					for _, s := range stateful {
						fmt.Printf("    %s (%s) → %s Motif + %s\n",
							s.Name, s.Image, s.Motif.MotifRef, s.Motif.AdminUI)
					}
				}
				if len(buildable) > 0 {
					fmt.Println()
					fmt.Printf("💡 Buildable services in %s: %s\n", composeFile, strings.Join(buildable, ", "))
					fmt.Println()
					fmt.Printf("  Run 'ork doctor init --use-compose %s' to generate per-app config\n", filepath.Base(info.ComposePath))
					printInternalURLHints(dir, buildable, cf)
				}
			}
		}

		// SMTP/Slack hint — shown before the "Orkestra will create" section.
		if info.HasSMTP || info.HasSlack {
			fmt.Println()
			fmt.Println("  ~ SMTP/Slack detected in .env")
			fmt.Println("    Run 'ork doctor init --name <app> --notify-me' to wire notifications.")
		}

		fmt.Println()
		fmt.Println("Orkestra will create:")

		varTxt := "variables"
		if len(info.Secrets) == 1 {
			varTxt = "variable"
		}

		if info.GitCommit == "" {
			info.GitCommit = "latest"
		}

		// If docker compose is the context
		if composeFile != "" && dockerfilePath == "" {
			dockerfilePath = composeFile
		}

		fmt.Printf("  %-22s image built from %s, tagged :%s\n", "Deployment", dockerfilePath, info.GitCommit)
		fmt.Printf("  %-22s with minimal RBAC to run your workloads\n", "Service Account")

		if len(info.Secrets) > 0 {
			fmt.Printf("  %-22s %s-secrets (%d %s from .env)\n",
				"Secret", info.Name, len(info.Secrets), varTxt)
		}

		if len(info.Config) > 0 {
			fmt.Printf("  %-22s %s-config  (%d %s from .env # ork:cfg)\n",
				"ConfigMap", info.Name, len(info.Config), varTxt)
		}

		fmt.Printf("  %-22s port %s\n", "Service", info.Port)

		if info.HasFrontend {
			fmt.Printf("  %-22s %s.local      (frontend detected)\n", "Ingress", info.Name)
		}

		if !noHA {
			fmt.Printf("  %-22s min 2 / max 10\n", "HPA")
			fmt.Printf("  %-22s minAvailable: 1 (if disruption happens)\n", "PDB")
		}

		if !noSecure {
			fmt.Printf("  %-22s enabled\n", "DeletionProtection")
		}
		fmt.Println()
		missing := 0
		fmt.Println("Missing dependencies:")
		if !doctor.KubectlAvailable() {
			fmt.Println("  kubectl   (will be installed during 'ork deploy')")
			missing++
		}
		if !doctor.HelmAvailable() {
			fmt.Println("  helm      (will be installed during 'ork deploy')")
			missing++
		}
		if missing == 0 {
			fmt.Print("  (none) ")
		}

		fmt.Println()
		fmt.Println()
		fmt.Println("Run 'ork doctor init --name <my-project>' to generate .orkestra/katalog.yaml")
		if !noHA {
			fmt.Println("Run with --no-ha to skip HPA and PDB (development mode)")
		}
		if !noSecure {
			fmt.Println("Run with --no-secure to skip deletion protection (development mode)")
		}

		return nil
	},
}

// doctorScanMultiApp scans each named subdirectory and prints per-app analysis.
func doctorScanMultiApp(baseDir string, appNames []string, noHA, noSecure bool) error {
	fmt.Println("\nExamining project...")
	fmt.Println()

	for _, name := range appNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		appDir := filepath.Join(baseDir, name)

		info, err := doctor.Detect(appDir)
		if err != nil {
			fmt.Printf("  %s: detection error: %v\n", name, err)
			continue
		}

		fmt.Printf("  %s/ (%s):\n", name, appDir)
		if info.HasDockerfile {
			fmt.Println("    ✓ Dockerfile found")
		} else {
			fmt.Println("    ✗ Dockerfile not found")
		}
		if info.Language != orktypes.LangUnknown {
			fmt.Printf("    ✓ Language: %s  (%s)\n", info.Language, info.LangMarker)
		} else {
			fmt.Println("    ~ Language: unknown")
		}
		fmt.Printf("    ✓ Port: %s\n", info.Port)
		if len(info.EnvVars) > 0 {
			fmt.Printf("    ✓ .env — %d variables\n", len(info.EnvVars))
		} else {
			fmt.Println("    ~ .env not found")
		}
		fmt.Println()
	}

	fmt.Println("Internal service URLs (set these in each app's .env before deploying):")
	for _, name := range appNames {
		name = strings.TrimSpace(name)
		appDir := filepath.Join(baseDir, name)
		info, err := doctor.Detect(appDir)
		port := "8080"
		if err == nil {
			port = info.Port
		}
		svcName := name + "-orkestra-svc"
		ns := name + "-orkestra-ns"
		envVar := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_URL"
		fmt.Printf("  %-20s http://%s.%s.svc.cluster.local:%s\n", envVar, svcName, ns, port)
	}
	fmt.Println()
	fmt.Printf("Run 'ork doctor init --app %s' to generate .orkestra/ config\n", strings.Join(appNames, ","))
	return nil
}

// printInternalURLHints prints the cluster-internal URLs for a set of compose services.
func printInternalURLHints(baseDir string, serviceNames []string, cf *doctor.ComposeFile) {
	if len(serviceNames) == 0 {
		return
	}
	fmt.Println("\nInternal service URLs (set these in each app's .env before deploying):")
	for _, name := range serviceNames {
		svc := cf.Services[name]
		port := "8080"
		// Try to extract port from compose ports field
		if svc.Ports != nil {
			if ports, ok := svc.Ports.([]interface{}); ok && len(ports) > 0 {
				p := fmt.Sprintf("%v", ports[0])
				// ports can be "host:container" or just "container"
				if idx := strings.LastIndex(p, ":"); idx >= 0 {
					port = p[idx+1:]
				} else {
					port = p
				}
			}
		}
		svcName := name + "-orkestra-svc"
		ns := name + "-orkestra-ns"
		envVar := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_URL"
		fmt.Printf("  %-20s http://%s.%s.svc.cluster.local:%s\n", envVar, svcName, ns, port)
	}
}

var doctorInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate .orkestra/ katalog and app config",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		noHA, _ := cmd.Flags().GetBool("no-ha")
		noSecure, _ := cmd.Flags().GetBool("no-secure")
		clean, _ := cmd.Flags().GetBool("clean")
		name, _ := cmd.Flags().GetString("name")
		addIngress, _ := cmd.Flags().GetBool("add-ingress")
		notifyMe, _ := cmd.Flags().GetBool("notify-me")
		useCompose, _ := cmd.Flags().GetString("use-compose")
		appFlag, _ := cmd.Flags().GetString("app")

		opts := doctor.GenerateOptions{
			NoHA:       noHA,
			NoSecure:   noSecure,
			Clean:      clean,
			AddIngress: addIngress,
			NotifyMe:   notifyMe,
			UseCompose: useCompose,
		}

		// ── Multi-app init (--app flag or --use-compose with multiple buildable services) ──
		if appFlag != "" || useCompose != "" {
			return doctorInitMultiApp(dir, cmd, opts, name, useCompose, appFlag)
		}

		// ── Single-app init (legacy) ──
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		opts.Name = name

		info, err := doctor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		if err := doctor.Init(info, opts); err != nil {
			return err
		}

		shouldUseCompose := useCompose != ""
		if err := buildx.WriteInitConfig(dir, shouldUseCompose, useCompose); err != nil {
			return fmt.Errorf("writing init config: %v", err)
		}

		crName := name + "-orkestra"
		ns := name + "-orkestra-ns"

		fmt.Println()
		fmt.Printf("App:       %s\n", name)
		fmt.Printf("AppConfig: %s\n", crName)
		fmt.Printf("Namespace: %s\n", ns)
		fmt.Println()
		fmt.Println("Generated .orkestra/katalog.yaml")
		fmt.Println("Generated .orkestra/app.yaml")
		if addIngress && !info.HasFrontend {
			fmt.Println("  (Ingress included via --add-ingress)")
		}
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Review .orkestra/katalog.yaml  (edit freely)")
		fmt.Println("  2. Fill in .orkestra/app.yaml      (replicas, host, controlCenterHost, etc.)")
		fmt.Println("  3. Run 'ork deploy --registry <your-registry>'")

		return nil
	},
}

// doctorInitMultiApp handles multi-app initialization from either --app or --use-compose.
func doctorInitMultiApp(baseDir string, cmd *cobra.Command, opts doctor.GenerateOptions, projectName, useCompose, appFlag string) error {
	orkBaseDir := filepath.Join(baseDir, ".orkestra")

	type appSpec struct {
		name       string
		dir        string
		dockerfile string
	}

	var apps []appSpec

	// perAppStateful holds the pre-computed stateful-service assignments for
	// each buildable app when initialising from a compose file. It is nil for
	// the --app flag path (no compose file, no stateful services to inject).
	var perAppStateful map[string][]doctor.StatefulService

	if useCompose != "" {
		// Derive apps from compose file's buildable services.
		composePath := useCompose
		if !filepath.IsAbs(composePath) {
			composePath = filepath.Join(baseDir, composePath)
		}
		cf, err := doctor.ParseCompose(composePath)
		if err != nil {
			return fmt.Errorf("reading compose file: %w", err)
		}
		buildable, allStateful := doctor.ClassifyServices(cf)
		if len(buildable) == 0 {
			return fmt.Errorf("no buildable services found in %s — add build: to services that need a Docker image", useCompose)
		}

		// Pre-compute which stateful services belong to each app.
		// Uses depends_on relationships; falls back to the first app for any
		// stateful service not referenced by any depends_on declaration.
		if len(allStateful) > 0 {
			perAppStateful = doctor.StatefulDepsPerApp(cf, buildable, allStateful)
		}

		for _, svcName := range buildable {
			svc := cf.Services[svcName]
			ctxRel, dockerfile := svc.BuildContext()
			absDir := baseDir
			if ctxRel != "" {
				if filepath.IsAbs(ctxRel) {
					absDir = ctxRel
				} else {
					absDir = filepath.Join(filepath.Dir(composePath), ctxRel)
				}
			}
			apps = append(apps, appSpec{name: svcName, dir: absDir, dockerfile: dockerfile})
		}
	} else {
		// Derive apps from --app flag
		for _, n := range strings.Split(appFlag, ",") {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			apps = append(apps, appSpec{name: n, dir: filepath.Join(baseDir, n)})
		}
	}

	if len(apps) == 0 {
		return fmt.Errorf("no apps to initialize")
	}

	fmt.Println()
	fmt.Println("Generating multi-app config...")
	fmt.Println()

	var initApps []buildx.AppEntry

	for _, app := range apps {
		appOrkDir := filepath.Join(orkBaseDir, app.name)
		appOpts := opts
		appOpts.Name = app.name
		appOpts.OutDir = appOrkDir

		if perAppStateful != nil {
			// Multi-app compose path: supply only the stateful services this
			// specific app depends on. Clearing UseCompose prevents doctor.Init
			// from re-parsing the compose file and injecting everything again.
			appOpts.UseCompose = ""
			appOpts.InjectStateful = perAppStateful[app.name] // nil = no stateful for this app
		}

		info, err := doctor.Detect(app.dir)
		if err != nil {
			return fmt.Errorf("detecting %s: %w", app.name, err)
		}

		if err := doctor.Init(info, appOpts); err != nil {
			return fmt.Errorf("init %s: %w", app.name, err)
		}

		fmt.Printf("  %s:\n", app.name)
		fmt.Printf("    Generated .orkestra/%s/katalog.yaml\n", app.name)
		fmt.Printf("    Generated .orkestra/%s/app.yaml\n", app.name)

		initApps = append(initApps, buildx.AppEntry{
			Name:       app.name,
			Dir:        app.dir,
			Dockerfile: app.dockerfile,
		})
	}

	// Persist init config
	cfg := buildx.InitConfig{
		UseCompose:  useCompose != "",
		ComposeFile: useCompose,
		Apps:        initApps,
	}
	if err := buildx.WriteInitConfigFull(baseDir, cfg); err != nil {
		return fmt.Errorf("writing init config: %w", err)
	}

	// Print internal URLs
	fmt.Println()
	fmt.Println("Internal service URLs (set these in each app's .env before deploying):")
	for _, app := range apps {
		appDir := app.dir
		info, _ := doctor.Detect(appDir)
		port := "8080"
		if info != nil {
			port = info.Port
		}
		svcName := app.name + "-orkestra-svc"
		ns := app.name + "-orkestra-ns"
		envVar := strings.ToUpper(strings.ReplaceAll(app.name, "-", "_")) + "_URL"
		fmt.Printf("  %-20s http://%s.%s.svc.cluster.local:%s\n", envVar, svcName, ns, port)
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review .orkestra/<app>/katalog.yaml for each app")
	fmt.Println("  2. Fill in .orkestra/<app>/app.yaml for each app")

	// When stateful services were wired to specific apps, tell the user exactly
	// where to find and configure the dependency keys.
	if len(perAppStateful) > 0 {
		fmt.Println()
		fmt.Println("  Stateful dependencies detected — configure in the app that owns each service:")
		for _, app := range apps {
			deps := perAppStateful[app.name]
			if len(deps) == 0 {
				continue
			}
			fmt.Printf("    .orkestra/%s/app.yaml\n", app.name)
			for _, dep := range deps {
				fmt.Printf("      ↳ %s  (e.g. %sImage, %sVolumeSize)\n",
					dep.Motif.MotifRef,
					dep.Motif.MotifRef,
					dep.Motif.MotifRef,
				)
			}
		}
	}

	fmt.Println()
	fmt.Println("  3. Run 'ork deploy --registry <your-registry>'")
	_ = projectName
	return nil
}

func init() {
	doctorCmd.PersistentFlags().Bool("no-ha", false, "Skip HPA and PDB (single replica)")
	doctorCmd.PersistentFlags().Bool("no-secure", false, "Skip deletion protection and protection labels")
	doctorCmd.PersistentFlags().Bool("clean", false, "Remove deletion protection webhook on operator shutdown")

	doctorCmd.Flags().String("app", "", "Comma-separated list of app subdirectories to scan (e.g. app,frontend)")

	doctorInitCmd.Flags().String("name", "", "App name (single-app mode)")
	doctorInitCmd.Flags().String("app", "", "Comma-separated list of app subdirectories (multi-app mode, e.g. app,frontend)")
	doctorInitCmd.Flags().Bool("add-ingress", false, "Include Ingress even when no frontend was auto-detected")
	doctorInitCmd.Flags().Bool("notify-me", false, "Auto-enable notifications using SMTP_*/SLACK_* from .env and your Git author")
	doctorInitCmd.Flags().String("use-compose", "", "Path to docker-compose.yaml — deploy all buildable services as separate apps")

	doctorCmd.AddCommand(doctorInitCmd)
	rootCmd.AddCommand(doctorCmd)

	// Shadow global flags so they don't appear under `ork init`
	doctorCmd.Flags().Bool("debug", false, "")
	doctorCmd.Flags().String("kubeconfig", "", "")
	doctorCmd.Flags().StringSlice("katalog", nil, "")
	doctorCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	doctorCmd.Flags().MarkHidden("debug")
	doctorCmd.Flags().MarkHidden("kubeconfig")
	doctorCmd.Flags().MarkHidden("katalog")
	doctorCmd.Flags().MarkHidden("verbose")
}
