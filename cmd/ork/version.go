package ork

import (
	"fmt"

	"github.com/spf13/cobra"
	// "github.com/spf13/viper"

	"github.com/ialexeze/orkestra/pkg/utils"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show Orkestra version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(utils.OrkestraLogoCLI)
		// fmt.Printf("Orkestra version: %s\n", viper.GetString("version"))
		fmt.Printf("Orkestra version: v%s\n", kfg.App().Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
