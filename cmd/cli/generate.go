//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/generate"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

// bundleOptsFromFor reads the --for flag and returns the corresponding BundleOptions.
// --for accepts a comma-separated list of component names: runtime (alias: run),
// gateway (alias: gw), cc (aliases: controlcenter, control-center).
// When --for is absent or empty, all three components are included (default).
func bundleOptsFromFor(cmd *cobra.Command) (generate.BundleOptions, error) {
	forVal, _ := cmd.Flags().GetString("for")
	if forVal == "" {
		return generate.DefaultBundleOptions(), nil
	}
	opts := generate.BundleOptions{}
	var unknown []string
	for _, part := range strings.Split(forVal, ",") {
		name := strings.TrimSpace(strings.ToLower(part))
		if name == "" {
			continue
		}
		switch name {
		case "run", "runtime":
			opts.IncludeRuntime = true
		case "gw", "gateway":
			opts.IncludeGateway = true
		case "cc", "controlcenter", "control-center":
			opts.IncludeControlCenter = true
		default:
			unknown = append(unknown, part)
		}
	}
	if len(unknown) > 0 {
		return generate.BundleOptions{}, fmt.Errorf(
			"orkestra: unknown --for value(s): %s\n\nValid values are:\n"+
				"  runtime   (alias: run)          — reconcilers, leader election\n"+
				"  gateway   (alias: gw)            — TLS, admission webhooks\n"+
				"  cc        (alias: controlcenter) — control-center\n\n"+
				"Example: --for gateway\n"+
				"         --for runtime,cc",
			strings.Join(unknown, ", "),
		)
	}
	if !opts.IncludeRuntime && !opts.IncludeGateway && !opts.IncludeControlCenter {
		return generate.BundleOptions{}, fmt.Errorf(
			"orkestra: --for produced an empty component list; nothing to generate\n\n" +
				"Valid values are: runtime (run), gateway (gw), cc (controlcenter, control-center)",
		)
	}
	return opts, nil
}

// defaultNamespace returns the namespace to use when --namespace is not supplied.
// Reads ORK_NAMESPACE from the environment so that CLI invocations inside
// an already-configured cluster automatically target the right namespace.
func defaultNamespace() string {
	if ns := os.Getenv("ORK_NAMESPACE"); ns != "" {
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

	expanded := parseFilePaths(katalogPaths)
	if len(expanded) == 0 {
		expanded = defaultFilePaths()
	}
	if len(expanded) == 0 {
		return nil, fmt.Errorf(errNoKatalog)
	}

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

		if err := generate.TypeRegistry(out.kat.Enabled(), dryRun); err != nil {
			return fmt.Errorf("generate runtime: %w", err)
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
	Short: "Generate RBAC ClusterRoles and ServiceAccounts for Orkestra components",
	Long: `Reads one or more katalog.yaml files, merges them, and generates minimal
ClusterRoles for the runtime and gateway processes, plus ServiceAccounts for
all three components (runtime, gateway, control center).

Use --for to limit the output to specific components. By default all three
are included. Multiple values are comma-separated.

Examples:
  ork generate rbac -f katalog.yaml
  ork generate rbac -f katalog.yaml --for gateway
  ork generate rbac -f katalog.yaml --for runtime,cc
  ork generate rbac -f a.yaml,b.yaml`,
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

		opts, err := bundleOptsFromFor(cmd)
		if err != nil {
			return err
		}

		runtimeRules := k.GenerateRuntimeRBACRules()
		gatewayRules := k.GenerateGatewayRBACRules()

		output, err := generate.RBACWithOptions(runtimeRules, gatewayRules, opts, namespace, outputFile)
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
  • ServiceAccounts for runtime, gateway, and control center
  • ClusterRoles and ClusterRoleBindings (one per process, minimal permissions)
  • ConfigMap embedding your Katalog

Use --for to limit the output to specific components. By default all three
are included. Multiple values are comma-separated.

Examples:
  ork generate bundle -f katalog.yaml
  ork generate bundle -f katalog.yaml --for gateway
  ork generate bundle -f katalog.yaml --for runtime,cc
  ork generate bundle -f katalog.yaml -o bundle.yaml -n orkestra-system`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := generateKatalog(cmd)
		if err != nil {
			return err
		}
		namespace, _ := cmd.Flags().GetString("namespace")
		workloadNamespace, _ := cmd.Flags().GetString("workload-namespace")
		outputFile, _ := cmd.Flags().GetString("output")

		k, err := katalog.BuildExpanded(kfg, out.m)
		if err != nil {
			return fmt.Errorf("build katalog: %w", err)
		}

		opts, err := bundleOptsFromFor(cmd)
		if err != nil {
			return err
		}

		log.Println("generating bundle...")

		runtimeRules := k.GenerateRuntimeRBACRules()
		gatewayRules := k.GenerateGatewayRBACRules()

		expanded, err := k.SerializeExpanded()
		if err != nil {
			return fmt.Errorf("serialize katalog: %w", err)
		}

		bundle, err := generate.RenderBundle(runtimeRules, gatewayRules, expanded, namespace, workloadNamespace, opts)
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

	// component-selection flag (shared by rbac and bundle)
	// --for runtime          → runtime SA + ClusterRole only
	// --for gateway          → gateway SA + ClusterRole only
	// --for runtime,gateway  → both, no CC SA
	// --for runtime,cc       → runtime + CC SA, no gateway
	// (absent)               → all three (default)
	for _, cmd := range []*cobra.Command{generateRbacCmd, generateBundleCmd} {
		cmd.Flags().String("for", "", "Limit output to specific components: runtime, gateway, cc (comma-separated; default: all)")
	}

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
