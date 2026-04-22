// cmd/cli/version.go
package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/orkspace/orkestra/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show Orkestra version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(utils.OrkestraLogoCLI)
		fmt.Printf("Orkestra version: %s\n", version.Short())
		fmt.Printf("Commit:           %s\n", version.Commit)
		fmt.Printf("Built:            %s\n", version.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Shadow global flags so they don't appear under `ork version`
	versionCmd.Flags().Bool("debug", false, "")
	versionCmd.Flags().String("kubeconfig", "", "")
	versionCmd.Flags().StringSlice("katalog", nil, "")
	versionCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	versionCmd.Flags().MarkHidden("debug")
	versionCmd.Flags().MarkHidden("kubeconfig")
	versionCmd.Flags().MarkHidden("katalog")
	versionCmd.Flags().MarkHidden("verbose")
}
