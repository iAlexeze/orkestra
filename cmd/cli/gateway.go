//go:build gateway

package cli

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/spf13/cobra"
)

var gatewayCmd = &cobra.Command{
	Use:   "gate",
	Short: "Start the Orkestra gateway (TLS + admission webhooks, cluster-only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(paths) == 0 {
			paths = defaultFilePaths()
		}
		if len(paths) == 0 {
			paths = kfg.Katalog().Paths()
		}
		if len(paths) == 0 {
			return fmt.Errorf(errNoKatalog)
		}

		m := merger.New(paths...)
		if err := m.Merge(); err != nil {
			return fmt.Errorf("merging katalogs: %w", err)
		}

		logger.Debug().
			Str("katalogs", strings.Join(paths, ", ")).
			Int("total", m.Count()).
			Int("enabled", m.EnabledCount()).
			Msg("katalogs merged")

		internal.KonductGateway(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
	gatewayCmd.Flags().StringSliceP("file", "f", nil, "Path(s) or URL(s) to crd-katalog.yaml (repeatable)")
}
