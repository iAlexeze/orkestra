//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/spf13/cobra"
)

// ── inspect ───────────────────────────────────────────────────────────────────

var inspectMotif bool

var inspectCmd = &cobra.Command{
	Use:   "inspect <name>:<version>",
	Short: "Show metadata for a pattern version",
	Args:  cobra.ExactArgs(1),
	Example: `  ork inspect postgres:v14
  ork inspect web-service:v1.0.0 --motif
  ork inspect oci://ghcr.io/myorg/patterns/redis:v7
  ork inspect redis:v1.0.0 --view katalog.yaml,simulate.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		kind := registry.KatalogKind
		if inspectMotif {
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

		info, err := client.Info(cmd.Context(), ref)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
				hint := fmt.Sprintf("\n\nhint: authenticate first:\n  docker login %s", ref.Registry)
				if !inspectMotif {
					hint += "\nhint: if this is a motif, re-run with --motif"
				}
				return fmt.Errorf("fetching info: %w%s", err, hint)
			}
			return fmt.Errorf("fetching info: %w", err)
		}

		// --view: skip the metadata block, just fetch and print requested files.
		viewArg, _ := cmd.Flags().GetString("view")
		if viewArg != "" {
			fileMap := make(map[string]registry.FileEntry, len(info.Files))
			available := make([]string, 0, len(info.Files))
			for _, f := range info.Files {
				fileMap[f.Name] = f
				available = append(available, f.Name)
			}
			for _, name := range strings.Split(viewArg, ",") {
				name = strings.TrimSpace(name)
				f, ok := fileMap[name]
				if !ok {
					fmt.Printf("  %s %q not in artifact (available: %s)\n", warningMark(), name, strings.Join(available, ", "))
					continue
				}
				fmt.Printf("── %s ──\n", name)
				data, err := client.ViewFile(cmd.Context(), ref, f)
				if err != nil {
					fmt.Printf("  error: %v\n", err)
					continue
				}
				fmt.Println(string(data))
			}
			return nil
		}

		m := info.Meta
		if m.Deprecated != nil {
			fmt.Printf("\n%s  This pattern is deprecated.\n", yellow("⚠"))
			if m.Deprecated.MigratedTo != "" {
				fmt.Printf("  Migrate to:  %s\n", bold(m.Deprecated.MigratedTo))
			}
			if m.Deprecated.Message != "" {
				fmt.Printf("  Note:        %s\n", m.Deprecated.Message)
			}
		}
		fmt.Printf("\n%s:%s\n", m.Name, m.Version)
		fmt.Printf("  Registry:    %s\n", ref.Registry)
		if m.Kind != "" {
			fmt.Printf("  Kind:        %s\n", m.Kind)
		}
		fmt.Printf("  Digest:      %s\n", info.Digest[:19]+"...")
		if !info.PushedAt.IsZero() {
			fmt.Printf("  Pushed:      %s\n", info.PushedAt.Format(time.RFC3339))
		}
		fmt.Printf("  Size:        %s\n", formatSize(info.Size))
		fmt.Printf("\n  Description: %s\n", wordWrap(m.Description, 55, "               "))
		if len(m.Tags) > 0 {
			fmt.Printf("  Tags:        %s\n", strings.Join(m.Tags, ", "))
		}
		if m.Author != "" {
			fmt.Printf("  Author:      %s\n", m.Author)
		}
		if m.License != "" {
			fmt.Printf("  License:     %s\n", m.License)
		}
		if m.Simulate != nil {
			switch m.Simulate.Status {
			case "passed":
				var suffix string
				if m.Simulate.Assertions == 1 {
					suffix = "1 assertion"
				} else if m.Simulate.Assertions > 1 {
					suffix = fmt.Sprintf("%d assertions", m.Simulate.Assertions)
				}
				if m.Simulate.Duration != "" {
					if suffix != "" {
						suffix += " · "
					}
					suffix += m.Simulate.Duration
				}
				if m.Simulate.TestedAt != "" {
					if t, err := time.Parse(time.RFC3339, m.Simulate.TestedAt); err == nil {
						if suffix != "" {
							suffix += " · "
						}
						suffix += "tested " + humanDuration(time.Since(t)) + " ago"
					}
				}
				fmt.Printf("  Simulate:    %s\n", simulateVerified(suffix))
			case "skipped":
				fmt.Printf("  Simulate:    %s\n", simulateSkipped())
			case "no-assertion":
				fmt.Printf("  Simulate:    %s\n", simulateNoAssertion())
			}
		}
		if m.Kind != registry.MotifKind {
			if m.E2E != nil {
				switch m.E2E.Status {
				case "passed":
					var suffix string
					if m.E2E.Assertions == 1 {
						suffix = "1 assertion"
					} else if m.E2E.Assertions > 1 {
						suffix = fmt.Sprintf("%d assertions", m.E2E.Assertions)
					}
					if m.E2E.Duration != "" {
						if suffix != "" {
							suffix += " · "
						}
						suffix += m.E2E.Duration
					}
					if m.E2E.TestedAt != "" {
						if t, err := time.Parse(time.RFC3339, m.E2E.TestedAt); err == nil {
							if suffix != "" {
								suffix += " · "
							}
							suffix += "tested " + humanDuration(time.Since(t)) + " ago"
						}
					}
					fmt.Printf("  E2E:         %s\n", e2eVerified(suffix))
				case "skipped":
					fmt.Printf("  E2E:         %s\n", e2eSkipped())
				}
			} else {
				fmt.Printf("  E2E:         %s\n", e2eNotVerified())
			}
		}
		if m.Typed != nil {
			parts := []string{}
			if m.Typed.HasHooks {
				parts = append(parts, "hooks")
			}
			if m.Typed.HasConstructor {
				parts = append(parts, "constructor")
			}
			fmt.Printf("  Typed:       %s · %s\n",
				green("✓ "+strings.Join(parts, ", ")),
				yellow("requires custom runtime image"),
			)
		}
		if len(info.Files) > 0 {
			fmt.Printf("\n  Files:\n")
			for _, f := range info.Files {
				fmt.Printf("    %-30s %s\n", f.Name, formatSize(f.Size))
			}
		}
		fmt.Printf("\nTo pull:\n")
		fmt.Printf("  ork pull %s:%s\n", m.Name, m.Version)
		fmt.Println()
		return nil
	},
}

func init() {
	inspectCmd.Flags().BoolVarP(&inspectMotif, "motif", "m", false, "Resolve as a motif (uses ORK_MOTIFS_REGISTRY)")
	inspectCmd.Flags().StringP("view", "v", "", "Comma-separated list of files to print before pulling (e.g. katalog.yaml,cr.yaml)")
	rootCmd.AddCommand(inspectCmd)
}
