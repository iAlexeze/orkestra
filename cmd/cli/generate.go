//go:build !runtime

package cli

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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

// parseFilePaths handles comma-separated values and returns a slice of paths
func parseFilePaths(paths []string) []string {
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
	katalogPaths, _ := cmd.Flags().GetStringSlice("file")

	if len(katalogPaths) == 0 {
		return nil, fmt.Errorf("--file is required (can be specified multiple times or as comma-separated values)")
	}

	expanded := parseFilePaths(katalogPaths)

	m := merger.New(expanded...)
	if err := m.Merge(); err != nil {
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

		if err := generate.RuntimeRegistry(out.kat.Enabled(), dryRun); err != nil {
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
  ork generate rbac --file ./website-katalog.yaml
  ork generate rbac --file a.yaml --file b.yaml
  ork generate rbac --file a.yaml,b.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		namespace, _ := cmd.Flags().GetString("namespace")
		outputFile, _ := cmd.Flags().GetString("output")

		log.Println("generating rbac...")

		k, err := katalog.BuildExpanded(kfg, out.m)
		if err != nil {
			return fmt.Errorf("build katalog: %w", err)
		}

		rules := k.GenerateRBACRules()

		output, err := generate.RBAC(rules, namespace, outputFile)
		if err != nil {
			return fmt.Errorf("generate rbac: %w", err)
		}

		return writeOutput(outputFile, "rbac.yaml", []byte(output))
	},
}

var generateConfigMapCmd = &cobra.Command{
	Use:   "configmap",
	Short: "Generate a ConfigMap embedding a Katalog or Komposer",
	Long: `Reads a katalog.yaml or komposer.yaml file and produces a ConfigMap
that embeds the file under data:<filename>. Useful for injecting Katalogs
into the in-cluster Orkestra runtime.

Example:
  ork generate configmap -f katalog.yaml
  ork generate configmap -f komposer.yaml -n orkestra-system -o out.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		namespace, _ := cmd.Flags().GetString("namespace")
		outputFile, _ := cmd.Flags().GetString("output")

		log.Println("generating configmap...")

		k, err := katalog.BuildExpanded(kfg, out.m)
		if err != nil {
			return fmt.Errorf("build katalog: %w", err)
		}
		expanded, err := k.SerializeExpanded()
		if err != nil {
			return fmt.Errorf("serialize katalog: %w", err)
		}

		cm, err := generate.ConfigMap(expanded, namespace)
		if err != nil {
			return fmt.Errorf("generate configmap: %w", err)
		}

		return writeOutput(outputFile, "config.yaml", cm)
	},
}

var generateBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Generate a complete installation bundle (RBAC + ConfigMap)",
	Long: `Generates a complete Orkestra installation bundle containing:
  • Namespace (default: 'orkestra-system')
  • ServiceAccounts (runtime + control center)
  • ClusterRole (minimal permissions derived from your Katalog)
  • ClusterRoleBinding
  • ConfigMap embedding your Katalog

The bundle is self-contained and ready to apply with kubectl.

Examples:
  ork generate bundle --file my-katalog.yaml
  ork generate bundle --file my-katalog.yaml -o bundle.yaml
  ork generate bundle --file my-katalog.yaml -o bundle/
  ork generate bundle --file my-katalog.yaml --namespace custom-ns`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the katalog paths as a slice
		katalogPaths, _ := cmd.Flags().GetStringSlice("file")
		if len(katalogPaths) == 0 {
			return fmt.Errorf("--file is required")
		}

		// Generate RBAC from merged Katalog
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}
		namespace, _ := cmd.Flags().GetString("namespace")
		workloadNamespace, _ := cmd.Flags().GetString("workload-namespace")
		outputFile, _ := cmd.Flags().GetString("output")

		log.Println("generating bundle...")

		k, err := katalog.BuildExpanded(kfg, out.m)
		if err != nil {
			return fmt.Errorf("build katalog: %w", err)
		}

		rules := k.GenerateRBACRules()

		expanded, err := k.SerializeExpanded()
		if err != nil {
			return fmt.Errorf("serialize katalog: %w", err)
		}

		bundle, err := generate.RenderBundle(rules, expanded, namespace, workloadNamespace)
		if err != nil {
			return fmt.Errorf("generate bundle: %w", err)
		}

		return writeOutput(outputFile, "bundle.yaml", []byte(bundle))

	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.AddCommand(generateKatalogCmd)
	generateCmd.AddCommand(generateCRDCmd)
	generateCmd.AddCommand(generateRegistryCmd)
	generateCmd.AddCommand(generateDocsCmd)
	generateCmd.AddCommand(generateDashboardsCmd)
	generateCmd.AddCommand(generateAllCmd)
	generateCmd.AddCommand(generateRbacCmd)
	generateCmd.AddCommand(generateConfigMapCmd)
	generateCmd.AddCommand(generateBundleCmd)

	// All three commands use StringSliceP so generateKatalog can read them uniformly.
	for _, cmd := range []*cobra.Command{generateConfigMapCmd, generateBundleCmd, generateRbacCmd} {
		cmd.Flags().StringSliceP("file", "f", []string{}, "Path to katalog.yaml or komposer.yaml (repeatable or comma-separated)")
	}

	generateRegistryCmd.Flags().StringP("dirs", "d", "", "Comma-separated list of project directories to generate registries for")
	generateRegistryCmd.Flags().Duration("fetch-timeout", 2*time.Minute, "Timeout to fetch Go hook or constructor from 'location'")

	// Shared flags for all file-consuming generate commands.
	for _, cmd := range []*cobra.Command{
		generateRegistryCmd,
		generateDocsCmd,
		generateDashboardsCmd,
		generateAllCmd,
		generateRbacCmd,
		generateConfigMapCmd,
		generateBundleCmd,
	} {
		cmd.Flags().Bool("dry-run", false, "Print generated output to stdout without writing files")
		cmd.Flags().StringP("output", "o", "", "Write generated output to file")
		cmd.Flags().StringP("namespace", "n", defaultNamespace(), "Namespace for the ServiceAccount")
	}

	// bundle-only flags
	generateBundleCmd.Flags().StringP("workload-namespace", "w", "", "Namespace for Deployment workloads (used by ork doctor deploy)")

	// Shadow global flags so they don't appear under `ork generate`
	generateCmd.Flags().Bool("debug", false, "")
	generateCmd.Flags().String("kubeconfig", "", "")
	generateCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	generateCmd.Flags().MarkHidden("debug")
	generateCmd.Flags().MarkHidden("kubeconfig")
	generateCmd.Flags().MarkHidden("verbose")

	cobra.MarkFlagRequired(generateCmd.Flags(), "katalog")
}
