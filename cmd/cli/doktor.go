//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/doktor"
	"github.com/spf13/cobra"
)

var doktorCmd = &cobra.Command{
	Use:   "doktor",
	Short: "Examine the project and show what Orkestra found",
	Long: `Examine the current directory and report what Orkestra discovered.

  ork doktor             show project analysis
  ork doktor init        generate .orkestra/katalog.yaml and .orkestra/app.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		fmt.Println("\nExamining project...")
		fmt.Println()

		info, err := doktor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		// --- Dockerfile ---
		if info.HasDockerfile {
			fmt.Println("  ✓ Dockerfile found")
		} else {
			fmt.Println("  ✗ Dockerfile not found — add one to build an image")
		}

		// --- Git ---
		if info.GitCommit != "" {
			fmt.Printf("  ✓ Git repository — commit: %s\n", info.GitCommit)
		} else {
			fmt.Println("  ✗ Not a git repository — run 'git init'")
		}

		// --- Language ---
		if info.Language != doktor.LangUnknown {
			fmt.Printf("  ✓ Language: %s  (%s)\n", info.Language, info.LangMarker)
		} else {
			fmt.Println("  ~ Language: unknown")
		}

		// --- Port ---
		fmt.Printf("  ✓ Port: %s\n", info.Port)

		// --- .env ---
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
				_, stateful := doktor.ClassifyServices(cf)
				if len(stateful) > 0 {
					fmt.Println()
					fmt.Printf("💡 Infrastructure services detected in %s:\n", composeFile)
					for _, s := range stateful {
						fmt.Printf("    %s (%s) → %s Motif + %s\n",
							s.Name, s.Image, s.Motif.MotifRef, s.Motif.AdminUI)
					}
					fmt.Println()
					fmt.Printf("  Run 'ork doktor init --name <app> --use-compose %s' to include them\n", filepath.Base(info.ComposePath))
				}
			}
		}

		// SMTP/Slack hint — shown before the "Orkestra will create" section.
		if info.HasSMTP || info.HasSlack {
			fmt.Println()
			fmt.Println("  ~ SMTP/Slack detected in .env")
			fmt.Println("    Run 'ork doktor init --name <app> --notify-me' to wire notifications.")
			fmt.Println("    Orkestra will create an orkestra-notification Secret and notify your team")
			fmt.Println("    on deployment status changes.")
		}

		fmt.Println()
		fmt.Println("Orkestra will create:")

		noHA, _ := cmd.Flags().GetBool("no-ha")
		noSecure, _ := cmd.Flags().GetBool("no-secure")
		if info.GitCommit == "" {
			info.GitCommit = "latest"
		}

		fmt.Printf("  Deployment     image built from Dockerfile, tagged :%s\n", info.GitCommit)
		if len(info.Secrets) > 0 {
			fmt.Printf("  Secret         %s-secrets (%d variables from .env)\n", info.AppName, len(info.Secrets))
		}
		if len(info.Config) > 0 {
			fmt.Printf("  ConfigMap      %s-config  (%d variables from .env # ork:cfg)\n", info.AppName, len(info.Config))
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

		// Detect missing dependencies
		fmt.Println()
		missing := 0
		fmt.Println("Missing dependencies:")
		if !doktor.KubectlAvailable() {
			fmt.Println("  kubectl   (will be installed during 'ork deploy')")
			missing += 1
		}
		if !doktor.HelmAvailable() {
			fmt.Println("  helm      (will be installed during 'ork deploy')")
			missing += 1
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

var doktorInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate .orkestra/katalog.yaml and .orkestra/app.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		info, err := doktor.Detect(dir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		noHA, _ := cmd.Flags().GetBool("no-ha")
		noSecure, _ := cmd.Flags().GetBool("no-secure")
		clean, _ := cmd.Flags().GetBool("clean")
		name, _ := cmd.Flags().GetString("name")
		addIngress, _ := cmd.Flags().GetBool("add-ingress")
		notifyMe, _ := cmd.Flags().GetBool("notify-me")
		useCompose, _ := cmd.Flags().GetString("use-compose")

		opts := doktor.GenerateOptions{
			NoHA:       noHA,
			NoSecure:   noSecure,
			Clean:      clean,
			Name:       name,
			AddIngress: addIngress,
			NotifyMe:   notifyMe,
			UseCompose: useCompose,
		}

		if err := doktor.Init(info, opts); err != nil {
			return err
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
		fmt.Println("Generated .orkestra/values.yaml")
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

func init() {
	doktorCmd.PersistentFlags().Bool("no-ha", false, "Skip HPA and PDB (single replica)")
	doktorCmd.PersistentFlags().Bool("no-secure", false, "Skip deletion protection and protection labels")
	doktorCmd.PersistentFlags().Bool("clean", false, "Remove deletion protection webhook on operator shutdown")

	doktorInitCmd.Flags().String("name", "", "App name (e.g. my-app → CR: my-app-orkestra, namespace: my-app-orkestra-ns)")
	doktorInitCmd.Flags().Bool("add-ingress", false, "Include Ingress even when no frontend was auto-detected")
	doktorInitCmd.Flags().Bool("notify-me", false, "Auto‑enable notifications using SMTP_*/SLACK_* from .env and your Git author")
	doktorInitCmd.Flags().String("use-compose", "", "Path to docker-compose.yaml — deploys all services including databases via Motifs")

	_ = doktorInitCmd.MarkFlagRequired("name")

	doktorCmd.AddCommand(doktorInitCmd)
	rootCmd.AddCommand(doktorCmd)
}
