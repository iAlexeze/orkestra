//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/orkspace/orkestra/pkg/e2e"
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
	registryPushE2EFile    string
	registryPushNoE2E      bool
)

var registryPushCmd = &cobra.Command{
	Use:   "push <name>:<version> <dir>  OR  push <dir>",
	Short: "Push a pattern or motif directory to the registry",
	Args:  cobra.RangeArgs(1, 2),
	Example: `  ork registry push postgres:v14 ./patterns/postgres/
  ork registry push redis:v7 ./motifs/redis/
  ORK_REGISTRY=oci://myregistry.io/patterns ork registry push payments:v1.0 ./payments/
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
			if _, err := katalog.BuildExpanded(kfg, m); err != nil {
				return fmt.Errorf("  ✗ %s: %w", registry.FileKatalog, err)
			}
			fmt.Printf("  %s %-20s valid\n", utils.SuccessMark(), registry.FileKatalog)

			if err := validateCRDFile(filepath.Join(dir, registry.FileCRD)); err != nil {
				return fmt.Errorf("  ✗ %s: %w", registry.FileCRD, err)
			}
			fmt.Printf("  %s %-20s valid\n", utils.SuccessMark(), registry.FileCRD)
		}

		for _, f := range files {
			if f == registry.FileKatalog || f == registry.FileCRD {
				continue // already printed above
			}
			info, _ := os.Stat(filepath.Join(dir, f))
			fmt.Printf("  %s %-20s (%s)\n", utils.SuccessMark(), f, formatSize(info.Size()))
		}

		// E2E gate: run e2e.yaml before pushing if it exists (Katalog only).
		// Skip with --force or --no-e2e.
		var e2eMeta *registry.PatternE2E
		if patternKind == registry.KatalogKind {
			e2eFile := registryPushE2EFile
			if e2eFile == "" {
				e2eFile = filepath.Join(dir, registry.FileE2E)
			} else if !filepath.IsAbs(e2eFile) {
				e2eFile = filepath.Join(dir, e2eFile)
			}
			if _, err := os.Stat(e2eFile); err == nil {
				if registryPushForce || registryPushNoE2E {
					fmt.Printf("  ~ E2E skipped\n")
					e2eMeta = &registry.PatternE2E{
						Status:   "skipped",
						TestedAt: time.Now().UTC().Format(time.RFC3339),
						Runner:   detectRunner(),
					}
				} else {
					fmt.Printf("\nRunning E2E gate (%s)...\n", registry.FileE2E)
					runner, err := e2e.New(e2eFile, "", false, false)
					if err != nil {
						return fmt.Errorf("e2e gate: %w\n\nUse --force or --no-e2e to skip", err)
					}
					result, err := runner.Run(cmd.Context())
					if err != nil {
						return fmt.Errorf("e2e gate failed: %w\n\nFix the test or use --force to push anyway", err)
					}
					fmt.Printf("  %s E2E passed (%s)\n", utils.SuccessMark(), result.Duration())
					e2eMeta = &registry.PatternE2E{
						Status:   "passed",
						Duration: result.Duration(),
						TestedAt: time.Now().UTC().Format(time.RFC3339),
						Runner:   detectRunner(),
					}
				}
			}
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		progress := func(file string, size int64) {
			fmt.Printf("  → %-20s (%s)\n", file, formatSize(size))
		}

		digest, err := client.Push(cmd.Context(), ref, dir, e2eMeta, progress)
		if err != nil {
			return fmt.Errorf("push failed: %w", err)
		}

		fmt.Printf("\n%s Pushed: %s\n", utils.SuccessMark(), ref.String())
		fmt.Printf("  Digest: %s\n", digest[:19]+"...")

		// If a pattern directory also contains motif.yaml, push it separately
		// to the motif registry so it can be imported standalone.
		if patternKind == registry.KatalogKind {
			motifYAML := filepath.Join(dir, registry.FileMotif)
			if _, err := os.Stat(motifYAML); err == nil {
				motifRef, err := registry.ResolveForKind(fmt.Sprintf("%s:%s", meta.Name, meta.Version), registry.MotifKind)
				if err == nil {
					fmt.Printf("\nAlso pushing %s to %s...\n", registry.FileMotif, motifRef.Registry)
					if mDigest, err := client.Push(cmd.Context(), motifRef, dir, nil, progress); err != nil {
						fmt.Fprintf(os.Stderr, "warning: motif push failed: %v\n", err)
					} else {
						fmt.Printf("%s Pushed motif: %s\n", utils.SuccessMark(), motifRef.String())
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
			fmt.Printf("  imports:\n")
			fmt.Printf("    registry:\n")
			fmt.Printf("      - url: %s\n", ref.String())
		}

		_ = meta
		return nil
	},
}

func detectRunner() string {
	switch {
	case os.Getenv("GITHUB_ACTIONS") == "true":
		return "github-actions"
	case os.Getenv("GITLAB_CI") == "true":
		return "gitlab-ci"
	case os.Getenv("CIRCLECI") == "true":
		return "circleci"
	case os.Getenv("JENKINS_URL") != "":
		return "jenkins"
	case os.Getenv("BUILDKITE") == "true":
		return "buildkite"
	case os.Getenv("DRONE") == "true":
		return "drone"
	case os.Getenv("CI") == "true":
		return "ci"
	default:
		return "local"
	}
}
