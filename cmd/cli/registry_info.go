//go:build !runtime

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/spf13/cobra"
)

// ── info ──────────────────────────────────────────────────────────────────────

var registryInfoCmd = &cobra.Command{
	Use:   "info <name>:<version>",
	Short: "Show metadata for a pattern version",
	Args:  cobra.ExactArgs(1),
	Example: `  ork registry info postgres:v14
  ork registry info oci://ghcr.io/myorg/patterns/redis:v7`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := registry.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		info, err := client.Info(cmd.Context(), ref)
		if err != nil {
			return fmt.Errorf("fetching info: %w", err)
		}

		m := info.Meta
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
		if m.E2E != nil {
			switch m.E2E.Status {
			case "passed":
				suffix := m.E2E.Duration
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
		fmt.Printf("\nTo pull:\n")
		fmt.Printf("  ork registry pull %s:%s\n", m.Name, m.Version)
		fmt.Println()
		return nil
	},
}
