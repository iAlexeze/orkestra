package cli

import (
	"github.com/ialexeze/orkestra/cmd/internal"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Orkestra runtime locally",
	Run: func(cmd *cobra.Command, args []string) {
		internal.Konduct(kfg, ctx)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
