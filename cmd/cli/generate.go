package cli

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/orkspace/orkestra/pkg/generate"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

// defaultNamespace returns the namespace to use when --namespace is not supplied.
// Reads ORKESTRA_NAMESPACE from the environment so that CLI invocations inside
// an already-configured cluster automatically target the right namespace.
func defaultNamespace() string {
	if ns := os.Getenv("ORKESTRA_NAMESPACE"); ns != "" {
		return ns
	}
	return "orkestra-system"
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Orkestra components",
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
	kat   *katalog.Katalog
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
		//return nil, fmt.Errorf("merge katalogs: %w", err)
		return nil, err
	}

	var kat katalog.Katalog
	kat.Spec = m.ToSpec()

	// Convert map to slice for generate functions
	crdMap := m.ToSpec().CRDs
	crds := make([]orktypes.CRDEntry, 0, len(crdMap))
	for _, c := range crdMap {
		crds = append(crds, c)
	}

	return &mergerOut{
		m:     m,
		crds:  crds,
		kat:   &kat,
		paths: katalogPaths,
	}, nil
}

var generateRuntimeCmd = &cobra.Command{
	Use:   "registry",
	Short: "Generate 'pkg/runtime/zz_generated_runtime_registry.go' from a Katalog (local or remote)",
	Long: `Reads one or more crd-katalog.yaml files (local paths or remote URLs), validates them,
and emits 'pkg/runtime/zz_generated_runtime_registry.go' containing RegisterRuntimeObjects() and
RegisterScheme() for all enabled CRDs with reconciler.default: true.

The file is created if it does not exist and overwritten on each run — idempotent.

Examples:
  ork generate registry --katalog ./example-crds/website-crd/website-katalog.yaml
  ork generate registry --katalog ./path/to/first.yaml --katalog ./path/to/second.yaml
  ork generate registry --katalog ./path/to/first.yaml,./path/to/second.yaml
  ork generate registry --katalog https://raw.githubusercontent.com/.../crd-katalog.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Printf("generating runtime registry...\n")
		log.Printf("dry-run: %t\n", dryRun)

		if err := generate.Runtime(out.m, dryRun); err != nil {
			return fmt.Errorf("generate runtime registry: %w", err)
		}

		log.Printf("runtime registry generated successfully\n")
		log.Printf("registry: %s/%s\n", generate.RuntimePackage, generate.RegistryFile)
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

		var k katalog.Katalog
		if _, err = k.KomposeKatalogFromYaml(kfg, out.m); err != nil {
			return fmt.Errorf("build katalog: %w", err)
		}
		if _, err = k.ValidateConfig(kfg); err != nil {
			return fmt.Errorf("validate katalog: %w", err)
		}

		rules := k.GenerateRBACRules()

		if err := generate.RBAC(kfg, rules, namespace, outputFile); err != nil {
			return fmt.Errorf("generate rbac: %w", err)
		}

		log.Println("rbac generated successfully")

		if outputFile != "" {
			log.Printf("out: %s\n", outputFile)
		}
		return nil
	},
}

var generateConfigMapCmd = &cobra.Command{
	Use:   "configmap",
	Short: "Generate a ConfigMap embedding a Katalog or Komposer",
	Long: `Reads a katalog.yaml or komposer.yaml file and produces a ConfigMap
that embeds the file under data:<filename>. Useful for injecting Katalogs
into the in-cluster Orkestra runtime.

Example:
  ork generate configmap -k katalog.yaml
  ork generate configmap -k komposer.yaml -n orkestra-system -o out.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the katalog file path directly, don't validate
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}

		namespace, _ := cmd.Flags().GetString("namespace")
		outputFile, _ := cmd.Flags().GetString("output")

		log.Println("generating configmap...")

		if err := generate.ConfigMap(katalogPath, namespace, outputFile); err != nil {
			return fmt.Errorf("generate configmap: %w", err)
		}

		log.Println("configmap generated successfully")
		if outputFile != "" {
			log.Printf("out: %s\n", outputFile)
		}
		return nil
	},
}

var generateBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Generate a complete installation bundle (RBAC + ConfigMap)",
	Long: `Generates a complete Orkestra installation bundle containing:
  • ServiceAccounts (runtime + control center)
  • ClusterRole (minimal permissions derived from your Katalog)
  • ClusterRoleBinding
  • ConfigMap embedding your Katalog

The bundle is self-contained and ready to apply with kubectl.

Examples:
  ork generate bundle --katalog my-katalog.yaml
  ork generate bundle --katalog my-katalog.yaml -o bundle.yaml
  ork generate bundle --katalog my-katalog.yaml --namespace custom-ns`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the katalog paths as a slice
		katalogPaths, _ := cmd.Flags().GetStringSlice("katalog")
		if len(katalogPaths) == 0 {
			return fmt.Errorf("--katalog is required")
		}

		// Use the first path for ConfigMap
		katalogPath := katalogPaths[0]

		// Generate RBAC from merged Katalog
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}
		namespace, _ := cmd.Flags().GetString("namespace")
		outputFile, _ := cmd.Flags().GetString("output")

		log.Println("generating bundle...")

		var k katalog.Katalog
		if _, err = k.KomposeKatalogFromYaml(kfg, out.m); err != nil {
			return fmt.Errorf("build katalog: %w", err)
		}
		if _, err = k.ValidateConfig(kfg); err != nil {
			return fmt.Errorf("validate katalog: %w", err)
		}

		rules := k.GenerateRBACRules()

		bundle, err := generate.RenderBundle(kfg, rules, katalogPath, namespace)
		if err != nil {
			return fmt.Errorf("generate bundle: %w", err)
		}

		log.Println("bundle generated successfully")

		if outputFile != "" {
			return os.WriteFile(outputFile, []byte(bundle), 0644)
		}

		fmt.Println(bundle)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.AddCommand(generateKatalogCmd)
	generateCmd.AddCommand(generateCRDCmd)
	generateCmd.AddCommand(generateRuntimeCmd)
	generateCmd.AddCommand(generateDocsCmd)
	generateCmd.AddCommand(generateDashboardsCmd)
	generateCmd.AddCommand(generateAllCmd)
	generateCmd.AddCommand(generateRbacCmd)
	generateCmd.AddCommand(generateConfigMapCmd)
	generateCmd.AddCommand(generateBundleCmd)

	// Register --katalog flag for commands that need it
	generateConfigMapCmd.Flags().StringP("katalog", "k", "", "Path to katalog.yaml or komposer.yaml")

	// For bundle, use StringSliceP to be compatible with generateKatalog
	generateBundleCmd.Flags().StringSliceP("katalog", "k", []string{}, "Path to katalog.yaml")

	generateRbacCmd.Flags().StringSliceP("katalog", "k", []string{}, "Path to katalog.yaml (can be specified multiple times or as comma-separated)")

	// Add shared flags
	for _, cmd := range []*cobra.Command{
		generateRuntimeCmd,
		generateDocsCmd,
		generateDashboardsCmd,
		generateAllCmd,
		generateRbacCmd,
	} {
		cmd.Flags().Bool("dry-run", false, "Print generated output to stdout without writing files")
		cmd.Flags().StringP("output", "o", "", "Write generated output to file")
		cmd.Flags().StringP("namespace", "n", defaultNamespace(), "Namespace for the ServiceAccount")
	}

	// Add shared flags for configmap and bundle (without StringSlice)
	for _, cmd := range []*cobra.Command{
		generateConfigMapCmd,
		generateBundleCmd,
	} {
		cmd.Flags().Bool("dry-run", false, "Print generated output to stdout without writing files")
		cmd.Flags().StringP("output", "o", "", "Write generated output to file")
		cmd.Flags().StringP("namespace", "n", defaultNamespace(), "Namespace for the ServiceAccount")
	}

	// Shadow global flags so they don't appear under `ork generate`
	generateCmd.Flags().Bool("debug", false, "")
	generateCmd.Flags().String("kubeconfig", "", "")
	generateCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	generateCmd.Flags().MarkHidden("debug")
	generateCmd.Flags().MarkHidden("kubeconfig")
	generateCmd.Flags().MarkHidden("verbose")
}
