package cli

import (
	"fmt"
	"log"

	"github.com/ialexeze/orkestra/pkg/generate"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Orkestra komponents",
}

var generateCRDCmd = &cobra.Command{
	Use:   "crd <name>",
	Short: "Generate a new CRD scaffold",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing CRD name")
			return
		}
		fmt.Printf("Generating CRD: %s\n", args[0])
		// TODO: scaffold API types
	},
}

var generateReconcilerCmd = &cobra.Command{
	Use:   "reconciler <name>",
	Short: "Generate a new reconciler scaffold",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing reconciler name")
			return
		}
		fmt.Printf("Generating reconciler: %s\n", args[0])
		// TODO: scaffold reconciler
	},
}

var generateRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Generate initialize/registry.go from a Katalog (local or remote)",
	Long: `Reads a crd-katalog.yaml (local path or remote URL), validates it,
and emits initialize/registry.go containing RegisterRuntimeObjects() and
RegisterScheme() for all enabled CRDs with reconciler.default: true.

The file is created if it does not exist and overwritten on each run — idempotent.

Examples:
  ork generate registry --katalog ./crdkatalog/crdkatalog.yaml
  ork generate registry --katalog https://raw.githubusercontent.com/.../crd-katalog.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Printf("generating registry...\n")
		log.Printf("katalog: %s\n", katalogPath)
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Registry(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate registry: %w", err)
		}

		log.Printf("registry generated successfully\n")
		log.Printf("registry: %s/%s\n", generate.RuntimePackage, generate.RegistryFile)
		log.Printf("hooks: %s/%s\n", generate.RuntimePackage, generate.HooksFile)
		return nil
	},
}

var generateDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate Markdown documentation for all CRDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Printf("generating docs...\n")
		log.Printf("katalog: %s\n", katalogPath)
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Docs(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate docs: %w", err)
		}

		logger.Info().Msg("docs generated successfully")
		log.Printf("out: %s\n", generate.DashDir)
		return nil
	},
}

var generateDashboardsCmd = &cobra.Command{
	Use:   "dashboards",
	Short: "Generate Grafana dashboards for all CRDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Println("generating dashboards...")
		log.Printf("katalog: %s\n", katalogPath)
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Dashboards(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate dashboards: %w", err)
		}

		log.Printf("dashboards generated successfully\n")
		log.Printf("out: %s\n", generate.DashDir)
		return nil
	},
}

var generateExamplesCmd = &cobra.Command{
	Use:   "examples",
	Short: "Generate example manifests for all CRDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		logger.Info().Str("katalog", katalogPath).Msg("generating examples...")
		log.Printf("katalog: %s\n", katalogPath)
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Examples(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate examples: %w", err)
		}

		log.Println("examples generated successfully")
		log.Printf("out: %s\n", generate.ExamplesDir)
		return nil
	},
}

var generateTestsCmd = &cobra.Command{
	Use:   "tests",
	Short: "Generate test scaffolding for all CRDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		logger.Info().Str("katalog", katalogPath).Msg("generating examples...")
		log.Printf("katalog: %s\n", katalogPath)
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Tests(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate tests: %w", err)
		}

		log.Println("tests generated successfully")
		log.Printf("out: %s\n", generate.TestDir)
		return nil
	},
}

var generateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Generate registry, docs, dashboards, examples, tests, and graphs",
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Println("running all generators...")

		if err := generate.Registry(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate registry: %w", err)
		}
		if err := generate.Docs(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate docs: %w", err)
		}
		if err := generate.Dashboards(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate dashboards: %w", err)
		}
		if err := generate.Examples(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate examples: %w", err)
		}
		if err := generate.Tests(katalogPath, dryRun); err != nil {
			return fmt.Errorf("generate tests: %w", err)
		}
		log.Println("all generators completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.AddCommand(generateCRDCmd)
	generateCmd.AddCommand(generateReconcilerCmd)
	generateCmd.AddCommand(generateRegistryCmd)
	generateCmd.AddCommand(generateDocsCmd)
	generateCmd.AddCommand(generateDashboardsCmd)
	generateCmd.AddCommand(generateExamplesCmd)
	generateCmd.AddCommand(generateTestsCmd)
	generateCmd.AddCommand(generateAllCmd)

	generateRegistryCmd.Flags().String("katalog", "", "Path or URL to crd-katalog.yaml (required)")
	generateRegistryCmd.Flags().Bool("dry-run", false, "Print generated output to stdout without writing files")
	generateDocsCmd.Flags().String("katalog", "", "Path or URL to crd-katalog.yaml (required)")
	generateDashboardsCmd.Flags().String("katalog", "", "Path or URL to crd-katalog.yaml (required)")
	generateExamplesCmd.Flags().String("katalog", "", "Path or URL to crd-katalog.yaml (required)")
	generateTestsCmd.Flags().String("katalog", "", "Path or URL to crd-katalog.yaml (required)")
	generateAllCmd.Flags().String("katalog", "", "Path or URL to crd-katalog.yaml (required)")
}
