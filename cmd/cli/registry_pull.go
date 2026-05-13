//go:build !runtime

package cli

import (
	"fmt"
	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

// ── pull ──────────────────────────────────────────────────────────────────────

var registryPullCmd = &cobra.Command{
	Use:   "pull <name>:<version>",
	Short: "Pull a pattern to the local cache",
	Args:  cobra.ExactArgs(1),
	Example: `  ork registry pull postgres:v14
  ork registry pull oci://ghcr.io/myorg/patterns/redis:v7
  ork registry pull postgres:v14 --refresh
  ork registry pull postgres:v14 --out ./postgres/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh, _ := cmd.Flags().GetBool("refresh")
		outDir, _ := cmd.Flags().GetString("out")

		ref, err := registry.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		if ref.IsCached() && !refresh {
			cacheDir, _ := ref.CachePath()
			fmt.Printf("  %s Already cached\n", utils.ColorGreen+"✓"+utils.ColorReset)
			fmt.Printf("  → %s\n", cacheDir)
			printPullSuggestions(ref, cacheDir)
			return nil
		}

		fmt.Printf("Pulling %s...\n  → %s\n", ref.ShortName(), ref.String())

		cacheDir, err := client.Pull(cmd.Context(), ref, refresh)
		if err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}

		// Copy to --out if requested
		if outDir != "" {
			if err := copyDir(cacheDir, outDir); err != nil {
				return fmt.Errorf("extracting to %s: %w", outDir, err)
			}
			fmt.Printf("  %s Extracted to %s\n", utils.ColorGreen+"✓"+utils.ColorReset, outDir)
			return nil
		}

		fmt.Printf("  %s Cached at %s\n", utils.ColorGreen+"✓"+utils.ColorReset, cacheDir)
		printPullSuggestions(ref, cacheDir)
		return nil
	},
}
