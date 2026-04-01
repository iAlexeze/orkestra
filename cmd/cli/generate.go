package cli

import (
	"fmt"
	"log"
	"strings"

	"github.com/ialexeze/orkestra/pkg/generate"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/merger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

const (
	DefaultNamespace = "orkestra-system"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Orkestra components",
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

// parseKatalogPaths handles comma-separated values and returns a slice of paths
func parseKatalogPaths(paths []string) []string {
	var expanded []string
	for _, p := range paths {
		// Split by comma and trim spaces
		parts := strings.Split(p, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				expanded = append(expanded, trimmed)
			}
		}
	}
	return expanded
}

type mergerOut struct {
	m     *merger.Merger
	crds  []orktypes.CRDEntry
	paths []string
}

func generateKatalog(cmd *cobra.Command) (*mergerOut, error) {
	katalogPaths, _ := cmd.Flags().GetStringSlice("katalog")

	if len(katalogPaths) == 0 {
		return nil, fmt.Errorf("--katalog is required (can be specified multiple times or as comma-separated values)")
	}

	expanded := parseKatalogPaths(katalogPaths)

	m := merger.New(expanded...)
	if err := m.Merge(); err != nil {
		return nil, fmt.Errorf("merge katalogs: %w", err)
	}

	return &mergerOut{
		m:     m,
		crds:  m.ToSpec().CRDs,
		paths: katalogPaths,
	}, nil
}

var generateRuntimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Generate 'pkg/runtime/zz_generated_runtime_registry.go' from a Katalog (local or remote)",
	Long: `Reads one or more crd-katalog.yaml files (local paths or remote URLs), validates them,
and emits 'pkg/runtime/zz_generated_runtime_registry.go' containing RegisterRuntimeObjects() and
RegisterScheme() for all enabled CRDs with reconciler.default: true.

The file is created if it does not exist and overwritten on each run — idempotent.

Examples:
  ork generate runtime --katalog ./example-crds/website-crd/website-katalog.yaml
  ork generate runtime --katalog ./path/to/first.yaml --katalog ./path/to/second.yaml
  ork generate runtime --katalog ./path/to/first.yaml,./path/to/second.yaml
  ork generate runtime --katalog https://raw.githubusercontent.com/.../crd-katalog.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Printf("generating runtime...\n")
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Runtime(out.m, dryRun); err != nil {
			return fmt.Errorf("generate runtime: %w", err)
		}

		log.Printf("runtime generated successfully\n")
		log.Printf("runtime: %s/%s\n", generate.RuntimePackage, generate.RegistryFile)
		return nil
	},
}

var generateDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate Markdown documentation for all CRDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Printf("generating docs...\n")
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Docs(out.crds, dryRun); err != nil {
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
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Println("generating dashboards...")
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Dashboards(out.crds, dryRun); err != nil {
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
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Printf("generating examples...\n")
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Examples(out.crds, dryRun); err != nil {
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
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Printf("generating tests...\n")
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Tests(out.crds, dryRun); err != nil {
			return fmt.Errorf("generate tests: %w", err)
		}

		log.Println("tests generated successfully")
		log.Printf("out: %s\n", generate.TestDir)
		return nil
	},
}

var generateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Generate runtime, docs, dashboards, examples, tests, and graphs",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return fmt.Errorf("merge katalogs: %w", err)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Println("running all generators...")

		if err := generate.Runtime(out.m, dryRun); err != nil {
			return fmt.Errorf("generate runtime: %w", err)
		}
		if err := generate.Docs(out.crds, dryRun); err != nil {
			return fmt.Errorf("generate docs: %w", err)
		}
		if err := generate.Dashboards(out.crds, dryRun); err != nil {
			return fmt.Errorf("generate dashboards: %w", err)
		}
		if err := generate.Examples(out.crds, dryRun); err != nil {
			return fmt.Errorf("generate examples: %w", err)
		}
		if err := generate.Tests(out.crds, dryRun); err != nil {
			return fmt.Errorf("generate tests: %w", err)
		}
		log.Println("all generators completed successfully")
		return nil
	},
}

var generateRbacCmd = &cobra.Command{
	Use:   "rbac",
	Short: "Generate RBAC ClusterRole for all CRDs in the Katalog",
	Long: `Reads one or more katalog.yaml files, merges them, and generates a minimal
ClusterRole containing only the RBAC rules required by the declared CRDs,
including conditional webhook permissions when validation, mutation, or
conversion rules are present.

Example:
  ork generate rbac --katalog ./website-katalog.yaml
  ork generate rbac --katalog a.yaml --katalog b.yaml
  ork generate rbac --katalog a.yaml,b.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		namespace, _ := cmd.Flags().GetString("namespace")
		outputFile, _ := cmd.Flags().GetString("output")

		log.Println("generating rbac...")

		if err := generate.RBAC(out.m, namespace, outputFile); err != nil {
			return fmt.Errorf("generate rbac: %w", err)
		}

		log.Println("rbac generated successfully")

		if outputFile != "" {
			log.Printf("out: %s\n", outputFile)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.AddCommand(generateCRDCmd)
	generateCmd.AddCommand(generateReconcilerCmd)
	generateCmd.AddCommand(generateRuntimeCmd)
	generateCmd.AddCommand(generateDocsCmd)
	generateCmd.AddCommand(generateDashboardsCmd)
	generateCmd.AddCommand(generateExamplesCmd)
	generateCmd.AddCommand(generateTestsCmd)
	generateCmd.AddCommand(generateAllCmd)
	generateCmd.AddCommand(generateRbacCmd)

	// Add flags to all commands that need katalog
	for _, cmd := range []*cobra.Command{
		generateRuntimeCmd,
		generateDocsCmd,
		generateDashboardsCmd,
		generateExamplesCmd,
		generateTestsCmd,
		generateAllCmd,
		generateRbacCmd,
	} {
		cmd.Flags().StringSliceP("katalog", "k", []string{}, "Path(s) or URL(s) to katalog.yaml (required, can be specified multiple times or as comma-separated values)")
		cmd.Flags().Bool("dry-run", false, "Print generated output to stdout without writing files")
		cmd.Flags().StringP("output", "o", "", "Write generated output to file")
		cmd.Flags().StringP("namespace", "n", DefaultNamespace, "Namespace for the ServiceAccount")
	}

	// Mark katalog as required for all commands
	for _, cmd := range []*cobra.Command{
		generateRuntimeCmd,
		generateDocsCmd,
		generateDashboardsCmd,
		generateExamplesCmd,
		generateTestsCmd,
		generateAllCmd,
	} {
		cobra.MarkFlagRequired(cmd.Flags(), "katalog")
	}
}
