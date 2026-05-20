//go:build !runtime

package cli

import "github.com/spf13/cobra"

func init() {
	// SilenceUsage is an option to silence usage when an error occurs.
	rootCmd.SilenceUsage = true
	// SilenceErrors is an option to quiet errors down stream.
	rootCmd.SilenceErrors = true

	rootCmd.CompletionOptions.DisableDefaultCmd = false

	// Dev-only persistent flags — not needed in the production runtime binary.
	rootCmd.PersistentFlags().String("kubeconfig", "", "Path to kubeconfig file")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Show full context")

	// Mark --file required for commands that need a Katalog file.
	// These variables only exist when built without -tags runtime, so this
	// block must live here rather than in root.go.
	for _, cmd := range []*cobra.Command{
		validateCmd,
		templateCmd,
		generateRegistryCmd,
		generateDashboardsCmd,
		generateAllCmd,
	} {
		cobra.MarkFlagRequired(cmd.Flags(), "file")
	}
}
