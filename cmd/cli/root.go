package cli

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/utils"
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

	// GLOBAL FLAGS
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
}

func initConfig() {
	viper.SetEnvPrefix("orkestra")
	viper.AutomaticEnv()

	debug, _ := rootCmd.Flags().GetBool("debug")
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
}
