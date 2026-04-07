package cli

import (
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/cmd/internal"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/merger"
	"github.com/spf13/cobra"
)

// The --katalog flag accepts one or more paths.
// Each path points to a Katalog file that may itself declare sources.
// The merger handles everything from there.

// Flag definition — same flag, now accepts multiple values:
//   cmd.Flags().StringSlice("katalog", nil,
//       "Path(s) or URL(s) to crd-katalog.yaml (repeatable)")

// Usage examples — all of these work identically:
//   ork run --katalog ./katalog.yaml
//   ork run --katalog ./project.yaml --katalog ./namespace.yaml
//   ork run --katalog https://remote/katalog.yaml
//   ork run --katalog ./local.yaml --katalog https://remote/extra.yaml

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Orkestra operator runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := cmd.Flags().GetStringSlice("katalog")
		if len(paths) == 0 {
			// fallback to env
			paths = kfg.Katalog().Paths
			if len(paths) == 0 {
				return fmt.Errorf("--katalog is required or set 'KATALOG_PATH' variable")
			}
		}

		m := merger.New(paths...)
		if err := m.Merge(); err != nil {
			return fmt.Errorf("merging katalogs: %w", err)
		}

		logger.Info().
			Str("katalogs", strings.Join(paths, ", ")).
			Int("total", m.Count()).
			Int("enabled", m.EnabledCount()).
			Msg("katalogs merged")

		internal.Konduct(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
