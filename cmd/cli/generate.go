package cli

import (
	"fmt"

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
  ork generate registry --katalog ./initialize/crd-katalog.yaml
  ork generate registry --katalog https://raw.githubusercontent.com/.../crd-katalog.yaml`,

	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPath, _ := cmd.Flags().GetString("katalog")
		if katalogPath == "" {
			return fmt.Errorf("--katalog is required")
		}

		logger.Info().Str("katalog", katalogPath).Msg("generating registry...")

		if err := generate.Registry(katalogPath); err != nil {
			return fmt.Errorf("generate registry: %w", err)
		}

		logger.Info().Str("out", "initialize/registry.go").Msg("registry generated successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateCRDCmd)
	generateCmd.AddCommand(generateReconcilerCmd)
	generateCmd.AddCommand(generateRegistryCmd)

	generateRegistryCmd.Flags().String("katalog", "", "Path or URL to crd-katalog.yaml (required)")
}
