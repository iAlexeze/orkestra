//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/spf13/cobra"
)

// ── list ──────────────────────────────────────────────────────────────────────

var registryListCmd = &cobra.Command{
	Use:   "list [registry-url]",
	Short: "List available patterns in the registry",
	Args:  cobra.MaximumNArgs(1),
	Example: `  ork registry list
  ork registry list --motifs
  ork registry list --katalogs
  ork registry list oci://ghcr.io/mycompany/patterns
  ork registry list --tag database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		onlyKatalogs, _ := cmd.Flags().GetBool("katalogs")
		onlyMotifs, _ := cmd.Flags().GetBool("motifs")

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		var entries []registry.PatternEntry
		var latestUpdatedAt string

		if len(args) > 0 {
			// Explicit registry URL — fetch only that one.
			idx, err := client.List(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("listing katalogs: %w", err)
			}
			entries = idx.Entries
			latestUpdatedAt = idx.UpdatedAt
		} else {
			// Fetch katalogs unless --motifs only.
			if !onlyMotifs {
				patURL := os.Getenv(registry.EnvPatternRegistry)
				if patURL == "" {
					patURL = registry.DefaultPatternRegistry
				}
				idx, _ := client.List(cmd.Context(), patURL)
				if idx != nil {
					entries = append(entries, idx.Entries...)
					if idx.UpdatedAt > latestUpdatedAt {
						latestUpdatedAt = idx.UpdatedAt
					}
				}
			}
			// Fetch motifs unless --katalogs only.
			if !onlyKatalogs {
				motifURL := os.Getenv(registry.EnvMotifRegistry)
				if motifURL == "" {
					motifURL = registry.DefaultMotifRegistry
				}
				idx, _ := client.List(cmd.Context(), motifURL)
				if idx != nil {
					entries = append(entries, idx.Entries...)
					if idx.UpdatedAt > latestUpdatedAt {
						latestUpdatedAt = idx.UpdatedAt
					}
				}
			}
		}

		label := "Orkestra Registry"
		switch {
		case onlyKatalogs:
			label = "Orkestra Katalogs"
		case onlyMotifs:
			label = "Orkestra Motifs"
		}
		fmt.Printf("\n%s\n", label)
		fmt.Printf("%s\n", strings.Repeat("─", 57))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tLATEST\tKIND\tE2E\tTAGS\tDESCRIPTION")

		count := 0
		for _, e := range entries {
			if tag != "" && !containsTag(e.Tags, tag) {
				continue
			}
			k := e.Kind
			if onlyKatalogs && k != registry.KatalogKind.ToString() {
				continue
			}
			if onlyMotifs && k != registry.MotifKind.ToString() {
				continue
			}
			tags := strings.Join(e.Tags, ", ")
			if len(tags) > 22 {
				tags = tags[:19] + "..."
			}
			desc := e.Description
			if len(desc) > 30 {
				desc = desc[:27] + "..."
			}
			e2eBadge := "-"
			switch e.E2EStatus {
			case "passed":
				e2eBadge = "✓"
			case "skipped":
				e2eBadge = "~"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", e.Name, e.LatestVersion, k, e2eBadge, tags, desc)
			count++
		}
		w.Flush()

		updatedAt := ""
		if latestUpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, latestUpdatedAt); err == nil {
				updatedAt = "  ·  Updated " + humanDuration(time.Since(t)) + " ago"
			}
		}
		noun := "patterns"
		if count == 1 {
			noun = "pattern"
		}
		fmt.Printf("\n%d %s%s\n", count, noun, updatedAt)

		fmt.Printf("\nTo pull:\n  ork registry pull <name>:<version>\n")
		fmt.Printf("\nTo filter:\n  ork registry list --katalogs\n  ork registry list --motifs\n")
		fmt.Println()
		return nil
	},
}
