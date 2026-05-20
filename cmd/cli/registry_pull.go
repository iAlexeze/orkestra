//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/motif"
	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

// ── pull ──────────────────────────────────────────────────────────────────────

var registryPullCmd = &cobra.Command{
	Use:   "pull [<name>:<version>]",
	Short: "Pull a pattern to the local cache",
	Args:  cobra.RangeArgs(0, 1),
	Example: `  ork registry pull postgres:v14
  ork registry pull oci://ghcr.io/myorg/patterns/redis:v7
  ork registry pull -f katalog.yaml
  ork registry pull -f komposer.yaml
  ork registry pull postgres:v14 --refresh`,
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh, _ := cmd.Flags().GetBool("refresh")
		outDir, _ := cmd.Flags().GetString("out")
		filePath, _ := cmd.Flags().GetString("file")

		// ── -f mode: pull all OCI imports from a katalog or komposer file ──────
		if filePath != "" {
			return pullFromFile(cmd, filePath, refresh)
		}

		if len(args) == 0 {
			return fmt.Errorf("provide a reference (e.g. postgres:v14) or --file <katalog.yaml>")
		}

		// ── single ref mode ────────────────────────────────────────────────────
		ref, err := registry.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		if ref.IsCached() && !refresh {
			cacheDir, _ := ref.CachePath()
			fmt.Printf("  %s Already cached\n", utils.SuccessMark())
			fmt.Printf("  → %s\n", cacheDir)
			printPullSuggestions(ref, cacheDir)
			return nil
		}

		fmt.Printf("Pulling %s...\n  → %s\n", ref.ShortName(), ref.String())

		cacheDir, err := client.Pull(cmd.Context(), ref, refresh)
		if err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}

		if outDir != "" {
			if err := copyDir(cacheDir, outDir); err != nil {
				return fmt.Errorf("extracting to %s: %w", outDir, err)
			}
			fmt.Printf("  %s Extracted to %s\n", utils.SuccessMark(), outDir)
			return nil
		}

		fmt.Printf("  %s Cached at %s\n", utils.SuccessMark(), cacheDir)
		printPullSuggestions(ref, cacheDir)
		return nil
	},
}

// pullFromFile extracts all OCI refs from a katalog or komposer file and pulls
// each one. Motif imports (Katalog) and registry imports (Komposer) are both
// handled, including bare-name shorthands without an oci:// prefix.
func pullFromFile(cmd *cobra.Command, filePath string, refresh bool) error {
	imports, err := registry.ExtractOCIImports(filePath)
	if err != nil {
		return fmt.Errorf("reading imports from %s: %w", filePath, err)
	}
	if imports.Empty() {
		fmt.Printf("  %s No OCI imports found in %s\n", utils.SuccessMark(), filePath)
		return nil
	}

	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("initializing client: %w", err)
	}

	var errs []string

	// Pull motif imports
	for _, imp := range imports.MotifImports {
		fmt.Printf("Pulling motif %s...\n", imp.Motif)
		if err := motif.PullImport(&imp); err != nil {
			fmt.Printf("  %s %v\n", utils.FailureMark(), err)
			errs = append(errs, err.Error())
		} else {
			fmt.Printf("  %s %s\n", utils.SuccessMark(), imp.Motif)
		}
	}

	// Pull registry imports (Komposer)
	for _, src := range imports.RegistrySources {
		cleanURL, version := src.ResolvedURL()
		// Strip oci:// prefix for Resolve
		cleanURL = strings.TrimPrefix(cleanURL, "oci://")
		ref, err := registry.Resolve(cleanURL + ":" + version)
		if err != nil {
			fmt.Printf("  %s resolving %s: %v\n", utils.FailureMark(), src.URL, err)
			errs = append(errs, err.Error())
			continue
		}
		if ref.IsCached() && !refresh {
			fmt.Printf("  %s Already cached: %s\n", utils.SuccessMark(), ref.ShortName())
			continue
		}
		fmt.Printf("Pulling %s...\n  → %s\n", ref.ShortName(), ref.String())
		if _, err := client.Pull(cmd.Context(), ref, refresh); err != nil {
			fmt.Printf("  %s %v\n", utils.FailureMark(), err)
			errs = append(errs, err.Error())
		} else {
			fmt.Printf("  %s %s\n", utils.SuccessMark(), ref.ShortName())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d pull(s) failed:\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}
