//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/buildx"
	"github.com/orkspace/orkestra/pkg/doktor"
	"github.com/spf13/cobra"
)

var doktorCmd = &cobra.Command{
	Use:   "doktor",
	Short: "Examine the project and show what Orkestra found",
	Long: `Examine the current directory and report what Orkestra discovered.

  ork doktor                       show project analysis
  ork doktor --app app,frontend    show per-app analysis (multi-app project)
  ork doktor init                  generate .orkestra/ katalog and app config`,
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
			return doktorScanMultiApp(dir, strings.Split(appFlag, ","), noHA, noSecure)
		}

		// Single-app scan (default)
		fmt.Println("\nExamining project...")
		fmt.Println()

		info, err := doktor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		if info.HasDockerfile {
			fmt.Println("  ✓ Dockerfile found")
		} else {
			fmt.Println("  ✗ Dockerfile not found — add one to build an image")
		}

		if info.GitCommit != "" {
			fmt.Printf("  ✓ Git repository — commit: %s\n", info.GitCommit)
		} else {
			fmt.Println("  ✗ Not a git repository — run 'git init'")
		}

		if info.Language != doktor.LangUnknown {
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
			cf, cfErr := doktor.ParseCompose(info.ComposePath)
			if cfErr == nil {
				buildable, stateful := doktor.ClassifyServices(cf)
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
					fmt.Printf("  Run 'ork doktor init --use-compose %s' to generate per-app config\n", filepath.Base(info.ComposePath))
					printInternalURLHints(dir, buildable, cf)
				}
			}
		}

		// SMTP/Slack hint — shown before the "Orkestra will create" section.
		if info.HasSMTP || info.HasSlack {
			fmt.Println()
			fmt.Println("  ~ SMTP/Slack detected in .env")
			fmt.Println("    Run 'ork doktor init --name <app> --notify-me' to wire notifications.")
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
		fmt.Printf("  Deployment     image built from Dockerfile, tagged :%s\n", info.GitCommit)
		if len(info.Secrets) > 0 {
			fmt.Printf("  Secret         %s-secrets (%d %s from .env)\n", info.AppName, len(info.Secrets), varTxt)
		}
		if len(info.Config) > 0 {
			fmt.Printf("  ConfigMap      %s-config  (%d %s from .env # ork:cfg)\n", info.AppName, len(info.Config), varTxt)
		}
		fmt.Printf("  Service        port %s\n", info.Port)
		if info.HasFrontend {
			fmt.Printf("  Ingress        %s.local      (frontend detected)\n", info.AppName)
		}
		if !noHA {
			fmt.Println("  HPA            min 2 / max 10")
			fmt.Println("  PDB            minAvailable: 1")
		}
		if !noSecure {
			fmt.Println("  DeletionProtection  enabled")
		}

		fmt.Println()
		missing := 0
		fmt.Println("Missing dependencies:")
		if !doktor.KubectlAvailable() {
			fmt.Println("  kubectl   (will be installed during 'ork deploy')")
			missing++
		}
		if !doktor.HelmAvailable() {
			fmt.Println("  helm      (will be installed during 'ork deploy')")
			missing++
		}
		if missing == 0 {
			fmt.Print("  (none) ")
		}

		fmt.Println()
		fmt.Println()
		fmt.Println("Run 'ork doktor init --name <my-project>' to generate .orkestra/katalog.yaml")
		if !noHA {
			fmt.Println("Run with --no-ha to skip HPA and PDB (development mode)")
		}
		if !noSecure {
			fmt.Println("Run with --no-secure to skip deletion protection (development mode)")
		}

		return nil
	},
}

// doktorScanMultiApp scans each named subdirectory and prints per-app analysis.
func doktorScanMultiApp(baseDir string, appNames []string, noHA, noSecure bool) error {
	fmt.Println("\nExamining project...")
	fmt.Println()

	for _, name := range appNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		appDir := filepath.Join(baseDir, name)

		info, err := doktor.Detect(appDir)
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
		if info.Language != doktor.LangUnknown {
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
		info, err := doktor.Detect(appDir)
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
	fmt.Printf("Run 'ork doktor init --app %s' to generate .orkestra/ config\n", strings.Join(appNames, ","))
	return nil
}

// printInternalURLHints prints the cluster-internal URLs for a set of compose services.
func printInternalURLHints(baseDir string, serviceNames []string, cf *doktor.ComposeFile) {
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

var doktorInitCmd = &cobra.Command{
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

		opts := doktor.GenerateOptions{
			NoHA:       noHA,
			NoSecure:   noSecure,
			Clean:      clean,
			AddIngress: addIngress,
			NotifyMe:   notifyMe,
			UseCompose: useCompose,
		}

		// ── Multi-app init (--app flag or --use-compose with multiple buildable services) ──
		if appFlag != "" || useCompose != "" {
			return doktorInitMultiApp(dir, cmd, opts, name, useCompose, appFlag)
		}

		// ── Single-app init (legacy) ──
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		opts.Name = name

		info, err := doktor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		if err := doktor.Init(info, opts); err != nil {
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

// doktorInitMultiApp handles multi-app initialization from either --app or --use-compose.
func doktorInitMultiApp(baseDir string, cmd *cobra.Command, opts doktor.GenerateOptions, projectName, useCompose, appFlag string) error {
	orkBaseDir := filepath.Join(baseDir, ".orkestra")

	type appSpec struct {
		name       string
		dir        string
		dockerfile string
	}

	var apps []appSpec

	if useCompose != "" {
		// Derive apps from compose file's buildable services
		composePath := useCompose
		if !filepath.IsAbs(composePath) {
			composePath = filepath.Join(baseDir, composePath)
		}
		cf, err := doktor.ParseCompose(composePath)
		if err != nil {
			return fmt.Errorf("reading compose file: %w", err)
		}
		buildable, _ := doktor.ClassifyServices(cf)
		if len(buildable) == 0 {
			return fmt.Errorf("no buildable services found in %s — add build: to services that need a Docker image", useCompose)
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

		info, err := doktor.Detect(app.dir)
		if err != nil {
			return fmt.Errorf("detecting %s: %w", app.name, err)
		}

		if err := doktor.Init(info, appOpts); err != nil {
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
		info, _ := doktor.Detect(appDir)
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
	fmt.Println("  3. Run 'ork deploy --registry <your-registry>'")
	_ = projectName
	return nil
}

func init() {
	doktorCmd.PersistentFlags().Bool("no-ha", false, "Skip HPA and PDB (single replica)")
	doktorCmd.PersistentFlags().Bool("no-secure", false, "Skip deletion protection and protection labels")
	doktorCmd.PersistentFlags().Bool("clean", false, "Remove deletion protection webhook on operator shutdown")

	doktorCmd.Flags().String("app", "", "Comma-separated list of app subdirectories to scan (e.g. app,frontend)")

	doktorInitCmd.Flags().String("name", "", "App name (single-app mode)")
	doktorInitCmd.Flags().String("app", "", "Comma-separated list of app subdirectories (multi-app mode, e.g. app,frontend)")
	doktorInitCmd.Flags().Bool("add-ingress", false, "Include Ingress even when no frontend was auto-detected")
	doktorInitCmd.Flags().Bool("notify-me", false, "Auto-enable notifications using SMTP_*/SLACK_* from .env and your Git author")
	doktorInitCmd.Flags().String("use-compose", "", "Path to docker-compose.yaml — deploy all buildable services as separate apps")

	doktorCmd.AddCommand(doktorInitCmd)
	rootCmd.AddCommand(doktorCmd)
}
