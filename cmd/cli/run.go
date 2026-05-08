package cli

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/spf13/cobra"
)

// The --file flag accepts one or more paths.
// Each path points to a Katalog file that may itself declare sources.
// The merger handles everything from there.

// Flag definition — same flag, now accepts multiple values:
//   cmd.Flags().StringSlice("katalog", nil,
//       "Path(s) or URL(s) to crd-katalog.yaml (repeatable)")

// Usage examples — all of these work identically:
//   ork run --file ./katalog.yaml
//   ork run --file ./project.yaml --file ./namespace.yaml
//   ork run --file https://remote/katalog.yaml
//   ork run --file ./local.yaml --file https://remote/extra.yaml

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Orkestra operator runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(paths) == 0 {
			// fallback to env
			paths = kfg.Katalog().Paths
			if len(paths) == 0 {
				return fmt.Errorf("--file is required or set 'KATALOG_PATH' variable")
			}
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

		// This is where the actual operator starts.
		// The --dev logic will be injected via a wrapper in run_dev.go.
		internal.Konduct(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
