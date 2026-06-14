//go:build !runtime && !gateway

package cli

import "github.com/spf13/cobra"

// pullCmd is a root-level alias for ork registry pull — mirrors docker pull UX.
var pullCmd = &cobra.Command{
	Use:   "pull [<name>:<version>]",
	Short: "Pull a pattern to the local cache (alias for ork registry pull)",
	Args:  cobra.RangeArgs(0, 1),
	Example: `  ork pull postgres:v14
  ork pull redis:v7 --motif
  ork pull -f katalog.yaml`,
	RunE: registryPullCmd.RunE,
}

func init() {
	pullCmd.Flags().Bool("refresh", false, "Bypass local cache and re-pull from registry")
	pullCmd.Flags().StringP("out", "o", "", "Extract pulled pattern to this directory")
	pullCmd.Flags().StringP("file", "f", "", "Pull all OCI imports from a katalog or komposer file")
	pullCmd.Flags().BoolP("motif", "m", false, "Resolve as a motif (uses ORK_MOTIFS_REGISTRY)")
	rootCmd.AddCommand(pullCmd)
}
