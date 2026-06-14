//go:build !runtime && !gateway

package cli

import "github.com/spf13/cobra"

// pushCmd is a root-level alias for ork registry push — mirrors docker push UX.
var pushCmd = &cobra.Command{
	Use:   "push <name>:<version> <dir>  OR  push <dir>",
	Short: "Push a pattern or motif to the registry (alias for ork registry push)",
	Args:  cobra.RangeArgs(1, 2),
	Example: `  ork push postgres:v14 ./patterns/postgres/
  ork push redis:v7 ./motifs/redis/
  ork push .`,
	RunE: registryPushCmd.RunE,
}

func init() {
	pushCmd.Flags().BoolVar(&registryPushForce, "force", false, "Force push even if metadata.version differs from tag or e2e fails")
	pushCmd.Flags().BoolVar(&registryPushUpdateMeta, "update-meta", false, "Persist overridden metadata.version back to the primary file")
	pushCmd.Flags().StringVar(&registryPushE2EFile, "e2e", "", "Path to e2e spec file (default: e2e.yaml in pattern dir)")
	pushCmd.Flags().BoolVar(&registryPushNoE2E, "no-e2e", false, "Skip the e2e gate even if e2e.yaml is present")
	pushCmd.Flags().BoolVar(&registryPushNoSimulate, "no-simulate", false, "Skip the simulate gate even if simulate.yaml is present")
	rootCmd.AddCommand(pushCmd)
}
