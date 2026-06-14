//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/motif"
	"github.com/orkspace/orkestra/pkg/registry"
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
		isMotif, _ := cmd.Flags().GetBool("motif")
		kind := registry.KatalogKind
		if isMotif {
			kind = registry.MotifKind
		}
		ref, err := registry.ResolveForKind(args[0], kind)
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		if ref.IsCached() && !refresh {
			cacheDir, _ := ref.CachePath()
			fmt.Printf("  %s Already cached\n", successMark())
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
			fmt.Printf("  %s Extracted to %s\n", successMark(), outDir)
			return nil
		}

		fmt.Printf("  %s Cached at %s\n", successMark(), cacheDir)
		printPullSuggestions(ref, cacheDir)

		if !isMotif {
			pullMotifDeps(cacheDir)
		}
		return nil
	},
}

// pullMotifDeps reads the katalog.yaml in cacheDir and pulls any OCI motif
// imports it declares. Warnings are printed but do not fail the main pull.
func pullMotifDeps(katalogCacheDir string) {
	katalogFile := filepath.Join(katalogCacheDir, registry.FileKatalog)
	if _, err := os.Stat(katalogFile); err != nil {
		return // not a katalog or no katalog.yaml — nothing to do
	}

	imports, err := registry.ExtractOCIImports(katalogFile)
	if err != nil || len(imports.MotifImports) == 0 {
		return
	}

	// Check if imports is empty
	if imports.Empty() {
		fmt.Printf("  %s No OCI imports found in %s\n", successMark(), katalogFile)
		return
	}

	fmt.Printf("\nPulling motif dependencies...\n")
	for _, imp := range imports.MotifImports {
		if motif.PullImport(&imp) == nil {
			fmt.Printf("  %s %s\n", successMark(), imp.Motif)
		} else {
			fmt.Printf("  %s %s (pull failed — will retry on next use)\n", warningMark(), imp.Motif)
		}
	}
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
		fmt.Printf("  %s No OCI imports found in %s\n", successMark(), filePath)
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
			fmt.Printf("  %s %v\n", failureMark(), err)
			errs = append(errs, err.Error())
		} else {
			fmt.Printf("  %s %s\n", successMark(), imp.Motif)
		}
	}

	// Pull registry imports (Komposer)
	for _, src := range imports.RegistrySources {
		cleanURL, version := src.ResolvedURL()
		// Strip oci:// prefix for Resolve
		cleanURL = strings.TrimPrefix(cleanURL, "oci://")
		ref, err := registry.Resolve(cleanURL + ":" + version)
		if err != nil {
			fmt.Printf("  %s resolving %s: %v\n", failureMark(), src.URL, err)
			errs = append(errs, err.Error())
			continue
		}
		if ref.IsCached() && !refresh {
			fmt.Printf("  %s Already cached: %s\n", successMark(), ref.ShortName())
			continue
		}
		fmt.Printf("Pulling %s...\n  → %s\n", ref.ShortName(), ref.String())
		if _, err := client.Pull(cmd.Context(), ref, refresh); err != nil {
			fmt.Printf("  %s %v\n", failureMark(), err)
			errs = append(errs, err.Error())
		} else {
			fmt.Printf("  %s %s\n", successMark(), ref.ShortName())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d pull(s) failed:\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}
