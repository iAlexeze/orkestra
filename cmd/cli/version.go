// cmd/cli/version.go
package cli

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/utils"
	"github.com/ialexeze/orkestra/pkg/version"
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
}
