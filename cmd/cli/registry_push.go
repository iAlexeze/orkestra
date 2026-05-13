//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

// ── push ──────────────────────────────────────────────────────────────────────

var (
	registryPushForce      bool
	registryPushUpdateMeta bool
)

var registryPushCmd = &cobra.Command{
	Use:   "push <name>:<version> <dir>  OR  push <dir>",
	Short: "Push a pattern or motif directory to the registry",
	Args:  cobra.RangeArgs(1, 2),
	Example: `  ork registry push postgres:v14 ./patterns/postgres/
  ork registry push redis:v7 ./motifs/redis/
  ORKESTRA_REGISTRY=oci://myregistry.io/patterns ork registry push payments:v1.0 ./payments/
  ork registry push ./patterns/postgres/   # use metadata.name:metadata.version from the pattern`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			refArg string
			dirArg string
		)

		// Two forms:
		// 1) push <ref> <dir>
		// 2) push <dir>   (use metadata.name:metadata.version)
		if len(args) == 2 {
			refArg = args[0]
			dirArg = args[1]
		} else {
			// single arg: treat as directory
			refArg = "" // will be derived from metadata
			dirArg = args[0]
		}

		dir, err := filepath.Abs(dirArg)
		if err != nil {
			return err
		}

		// Auto-detect pattern kind before resolving the registry URL.
		patternKind, spec, files, err := registry.ValidatePatternDirectory(dir)
		if err != nil {
			return fmt.Errorf("\n  ✗ %w", err)
		}

		// Load metadata from the primary file
		meta, err := registry.LoadPatternMeta(dir, spec)
		if err != nil {
			return fmt.Errorf("reading metadata: %w", err)
		}

		// If user provided a ref, extract tag and validate against metadata.version
		var providedTag string
		if refArg != "" {
			providedTag = registry.ExtractTagVersion(refArg)
			// If providedTag is empty, user may have passed a registry path without tag.
			// We'll still resolve the ref below; but only validate if a tag was present.
		}

		// If user did not provide a ref, build one from metadata.name:metadata.version
		if refArg == "" {
			if meta.Name == "" {
				return fmt.Errorf("metadata.name is required in %s", spec.PrimaryFile)
			}
			if meta.Version == "" {
				meta.Version = "latest"
			}
			refArg = fmt.Sprintf("%s:%s", meta.Name, meta.Version)
			providedTag = meta.Version
		} else {
			// If user provided a tag and metadata has a non-empty version, ensure they match.
			if providedTag != "" && meta.Version != "" && meta.Version != providedTag {
				msg := fmt.Errorf("%s: metadata.version %q does not match provided tag %q; use '--force' to override", spec.PrimaryFile, meta.Version, providedTag)

				if registryPushForce {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v (continuing due to --force)\n", utils.Yellow("Warning"), msg)

					// persist if requested
					if registryPushUpdateMeta {
						if err := registry.PersistMetadataVersion(dir, spec.PrimaryFile, providedTag); err != nil {
							return fmt.Errorf("failed to update metadata in %s: %w", spec.PrimaryFile, err)
						}
						fmt.Fprintf(cmd.OutOrStdout(), "updated %s: metadata.version -> %q\n", spec.PrimaryFile, providedTag)
					}

					meta.Version = providedTag
				} else {
					return msg
				}
			}
			// If metadata.version is empty but user provided a tag, populate meta.Version so we remain consistent.
			if meta.Version == "" && providedTag != "" {
				meta.Version = providedTag
			}
		}

		// Resolve against the kind-specific default registry.
		ref, err := registry.ResolveForKind(refArg, patternKind)
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		printBanner()
		fmt.Printf("Pushing %s (%s) to %s...\n", refArg, patternKind, ref.Registry)

		// Kind-specific validation.
		if patternKind == registry.KatalogKind {
			m := merger.New(filepath.Join(dir, registry.FileKatalog))
			if err := m.Merge(); err != nil {
				return fmt.Errorf("  ✗ %s: %w", registry.FileKatalog, err)
			}
			var kat katalog.Katalog
			if _, err := kat.KomposeRuntimeKatalog(kfg, m); err != nil {
				return fmt.Errorf("  ✗ %s: %w", registry.FileKatalog, err)
			}
			if _, err := kat.ValidateConfig(kfg); err != nil {
				return fmt.Errorf("  ✗ %s: %w", registry.FileKatalog, err)
			}
			fmt.Printf("  %s %-20s valid\n", utils.ColorGreen+"✓"+utils.ColorReset, registry.FileKatalog)

			if err := validateCRDFile(filepath.Join(dir, registry.FileCRD)); err != nil {
				return fmt.Errorf("  ✗ %s: %w", registry.FileCRD, err)
			}
			fmt.Printf("  %s %-20s valid\n", utils.ColorGreen+"✓"+utils.ColorReset, registry.FileCRD)
		}

		for _, f := range files {
			if f == registry.FileKatalog || f == registry.FileCRD {
				continue // already printed above
			}
			info, _ := os.Stat(filepath.Join(dir, f))
			fmt.Printf("  %s %-20s (%s)\n", utils.ColorGreen+"✓"+utils.ColorReset, f, formatSize(info.Size()))
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		progress := func(file string, size int64) {
			fmt.Printf("  → %-20s (%s)\n", file, formatSize(size))
		}

		digest, err := client.Push(cmd.Context(), ref, dir, progress)
		if err != nil {
			return fmt.Errorf("push failed: %w", err)
		}

		fmt.Printf("\n%s Pushed: %s\n", utils.ColorGreen+"✓"+utils.ColorReset, ref.String())
		fmt.Printf("  Digest: %s\n", digest[:19]+"...")

		// If a pattern directory also contains motif.yaml, push it separately
		// to the motif registry so it can be imported standalone.
		if patternKind == registry.KatalogKind {
			motifYAML := filepath.Join(dir, registry.FileMotif)
			if _, err := os.Stat(motifYAML); err == nil {
				motifRef, err := registry.ResolveForKind(fmt.Sprintf("%s:%s", meta.Name, meta.Version), registry.MotifKind)
				if err == nil {
					fmt.Printf("\nAlso pushing %s to %s...\n", registry.FileMotif, motifRef.Registry)
					if mDigest, err := client.Push(cmd.Context(), motifRef, dir, progress); err != nil {
						fmt.Fprintf(os.Stderr, "warning: motif push failed: %v\n", err)
					} else {
						fmt.Printf("%s Pushed motif: %s\n", utils.ColorGreen+"✓"+utils.ColorReset, motifRef.String())
						fmt.Printf("  Digest: %s\n", mDigest[:19]+"...")
					}
				}
			}
		}

		fmt.Printf("\nTo import in a Katalog:\n")
		if patternKind == registry.MotifKind {
			fmt.Printf("  imports:\n")
			fmt.Printf("    - motif: %s\n", ref.String())
		} else {
			fmt.Printf("  sources:\n")
			fmt.Printf("    registry:\n")
			fmt.Printf("      - url: %s\n", ref.String())
		}

		_ = meta
		return nil
	},
}
