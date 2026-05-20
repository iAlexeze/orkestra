// run.go — Production runtime entrypoint.
//
// This command has one responsibility: run Orkestra.
// It performs exactly three tasks:
//  1. Load all provided katalog files.
//  2. Merge them into a single resolved Katalog.
//  3. Start the Orkestra runtime using the merged result.
//
// All development‑only behavior (cluster checks, dependency setup,
// Kind provisioning, extra flags) lives in run_dev.go and is excluded
// from production builds via build tags.

//go:build runtime

package cli

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/spf13/cobra"
)

// If -f is not provided, Orkestra reads katalog.yaml (or komposer.yaml) from
// the current directory — the same convention as Docker and Compose.
// Pass -f explicitly only when using a non-standard filename or multiple files.

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Orkestra Runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(paths) == 0 {
			paths = defaultFilePaths()
		}
		if len(paths) == 0 {
			paths = kfg.Katalog().Paths
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

		// This is where the actual operator starts.
		// The --dev logic will be injected via a wrapper in run_dev.go.
		internal.Konduct(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
