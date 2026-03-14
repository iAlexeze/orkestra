package cli

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	kfg *konfig.Konfig
	ctx context.Context
)

var rootCmd = &cobra.Command{
	Use:   "ork",
	Short: "Orkestra — The Universal CRD Runtime",
	Long: fmt.Sprintf(`
%s
Orkestra — The Universal CRD Runtime
Kompose. Konduct. OrKestrate.
`, utils.OrkestraLogoCLI),
}

func Execute(k *konfig.Konfig, c context.Context) {
	kfg = k
	ctx = c

	if err := rootCmd.Execute(); err != nil {
		utils.Exit(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetEnvPrefix("orkestra")
	viper.AutomaticEnv()
}
