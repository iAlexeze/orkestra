// cmd/cli/registry.go
//
// ork registry — push, pull, info, list
//
// All four commands follow the same pattern as ork notes and ork init:
// minimal flags, clear output, no hidden state.

//go:build !runtime

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Push, pull, and inspect Orkestra patterns from OCI registries",
	Long: `Manage Orkestra operator patterns in OCI registries.

  ork registry push <name>:<version> <dir>    push a pattern directory
  ork registry pull <name>:<version>          pull a pattern to local cache
  ork registry info <name>:<version>          show pattern metadata
  ork registry list [registry-url]            list available patterns

Authentication uses ~/.docker/config.json — run 'docker login' first.
Override the default registry with ORKESTRA_REGISTRY:

  export ORKESTRA_REGISTRY=oci://myregistry.internal/patterns`,
}

// ── push ──────────────────────────────────────────────────────────────────────

var registryPushCmd = &cobra.Command{
	Use:   "push <name>:<version> <dir>",
	Short: "Push a pattern directory to the registry",
	Args:  cobra.ExactArgs(2),
	Example: `  ork registry push postgres:v14 ./postgres/
  ork registry push mycompany/payments:v1.0.0 ./payments/
  ORKESTRA_REGISTRY=oci://myregistry.io/patterns ork registry push redis:v7 ./redis/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := registry.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}
		dir, err := filepath.Abs(args[1])
		if err != nil {
			return err
		}

		printBanner()
		fmt.Printf("Validating pattern...\n")

		meta, files, err := registry.ValidateDirectory(dir)
		if err != nil {
			return fmt.Errorf("\n  ✗ %w", err)
		}

		for _, f := range files {
			info, _ := os.Stat(filepath.Join(dir, f))
			fmt.Printf("  %s %-20s (%s)\n", utils.ColorGreen+"✓"+utils.ColorReset, f, formatSize(info.Size()))
		}

		fmt.Printf("\nPushing %s to %s...\n", meta.Name+":"+meta.Version, ref.Registry)

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
		fmt.Printf("\nReference this pattern in a Komposer:\n")
		fmt.Printf("  sources:\n")
		fmt.Printf("    registry:\n")
		fmt.Printf("      - url: %s\n", ref.String())
		return nil
	},
}

// ── pull ──────────────────────────────────────────────────────────────────────

var registryPullCmd = &cobra.Command{
	Use:   "pull <name>:<version>",
	Short: "Pull a pattern to the local cache",
	Args:  cobra.ExactArgs(1),
	Example: `  ork registry pull postgres:v14
  ork registry pull oci://ghcr.io/myorg/patterns/redis:v7
  ork registry pull postgres:v14 --refresh
  ork registry pull postgres:v14 --out ./postgres/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh, _ := cmd.Flags().GetBool("refresh")
		outDir, _ := cmd.Flags().GetString("out")

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
			fmt.Printf("  %s Already cached\n", utils.ColorGreen+"✓"+utils.ColorReset)
			fmt.Printf("  → %s\n", cacheDir)
			printPullSuggestions(ref, cacheDir)
			return nil
		}

		fmt.Printf("Pulling %s...\n  → %s\n", ref.ShortName(), ref.String())

		cacheDir, err := client.Pull(cmd.Context(), ref, refresh)
		if err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}

		// Copy to --out if requested
		if outDir != "" {
			if err := copyDir(cacheDir, outDir); err != nil {
				return fmt.Errorf("extracting to %s: %w", outDir, err)
			}
			fmt.Printf("  %s Extracted to %s\n", utils.ColorGreen+"✓"+utils.ColorReset, outDir)
			return nil
		}

		fmt.Printf("  %s Cached at %s\n", utils.ColorGreen+"✓"+utils.ColorReset, cacheDir)
		printPullSuggestions(ref, cacheDir)
		return nil
	},
}

func printPullSuggestions(ref *registry.Ref, cacheDir string) {
	fmt.Printf("\nTo use this pattern:\n")
	fmt.Printf("  ork run -k %s\n", filepath.Join(cacheDir, registry.FileKatalog))
	fmt.Printf("\nOr reference in a Komposer:\n")
	fmt.Printf("  sources:\n")
	fmt.Printf("    registry:\n")
	fmt.Printf("      - url: %s\n", ref.String())
}

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
		if len(m.Requires.Providers) > 0 {
			fmt.Printf("\n  Requires:\n")
			for _, p := range m.Requires.Providers {
				fmt.Printf("    - %s provider\n", p)
			}
		}
		if len(m.Changelog) > 0 {
			fmt.Printf("\n  Changelog:\n")
			for _, e := range m.Changelog {
				fmt.Printf("    %-10s %s\n", e.Version, e.Notes)
			}
		}
		fmt.Printf("\nTo pull:\n")
		fmt.Printf("  ork registry pull %s:%s\n", m.Name, m.Version)
		fmt.Printf("\nTo use in a Komposer:\n")
		fmt.Printf("  sources:\n")
		fmt.Printf("    registry:\n")
		fmt.Printf("      - url: %s\n", ref.String())
		fmt.Println()
		return nil
	},
}

// ── list ──────────────────────────────────────────────────────────────────────

var registryListCmd = &cobra.Command{
	Use:   "list [registry-url]",
	Short: "List available patterns in a registry",
	Args:  cobra.MaximumNArgs(1),
	Example: `  ork registry list
  ork registry list oci://ghcr.io/mycompany/patterns
  ork registry list --tag database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")

		registryURL := registry.DefaultRegistry
		if len(args) > 0 {
			registryURL = args[0]
		} else if env := os.Getenv(registry.EnvRegistry); env != "" {
			registryURL = env
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		index, err := client.List(cmd.Context(), registryURL)
		if err != nil {
			return fmt.Errorf("listing patterns: %w", err)
		}

		displayURL := strings.TrimPrefix(registryURL, "oci://")
		fmt.Printf("\nOrkestra Registry  (%s)\n", displayURL)
		fmt.Printf("%s\n", strings.Repeat("─", 57))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tLATEST\tTAGS\tDESCRIPTION")

		count := 0
		for _, p := range index.Patterns {
			if tag != "" && !containsTag(p.Tags, tag) {
				continue
			}
			tags := strings.Join(p.Tags, ", ")
			if len(tags) > 22 {
				tags = tags[:19] + "..."
			}
			desc := p.Description
			if len(desc) > 30 {
				desc = desc[:27] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.LatestVersion, tags, desc)
			count++
		}
		w.Flush()

		updatedAt := "unknown"
		if index.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, index.UpdatedAt); err == nil {
				updatedAt = humanDuration(time.Since(t)) + " ago"
			}
		}

		fmt.Printf("\n%d patterns  ·  %s  ·  Updated %s\n", count, displayURL, updatedAt)
		fmt.Printf("\nTo pull a pattern:\n  ork registry pull <name>:<version>\n")
		if os.Getenv(registry.EnvRegistry) == "" {
			fmt.Printf("\nTo use a different registry:\n  export %s=oci://myregistry.internal/patterns\n", registry.EnvRegistry)
		}
		fmt.Println()
		return nil
	},
}

// ── registration ──────────────────────────────────────────────────────────────

func init() {
	rootCmd.AddCommand(registryCmd)
	registryCmd.AddCommand(registryPushCmd)
	registryCmd.AddCommand(registryPullCmd)
	registryCmd.AddCommand(registryInfoCmd)
	registryCmd.AddCommand(registryListCmd)

	registryPullCmd.Flags().Bool("refresh", false, "Bypass local cache and re-pull from registry")
	registryPullCmd.Flags().StringP("out", "o", "", "Extract pulled pattern to this directory")

	registryListCmd.Flags().StringP("tag", "t", "", "Filter by tag (e.g. database, stateful, security)")

	// Shadow global flags
	for _, cmd := range []*cobra.Command{registryCmd, registryPushCmd, registryPullCmd, registryInfoCmd, registryListCmd} {
		cmd.Flags().Bool("debug", false, "")
		cmd.Flags().String("kubeconfig", "", "")
		cmd.Flags().StringSlice("katalog", nil, "")
		cmd.Flags().Bool("verbose", false, "")
		cmd.Flags().MarkHidden("debug")
		cmd.Flags().MarkHidden("kubeconfig")
		cmd.Flags().MarkHidden("katalog")
		cmd.Flags().MarkHidden("verbose")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func wordWrap(s string, width int, indent string) string {
	if len(s) <= width {
		return s
	}
	return s[:width] + "\n" + indent + wordWrap(s[width:], width, indent)
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// withContext wraps RunE to inject a context — used for cancellation.
func withContext(ctx context.Context, fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(ctx)
		return fn(cmd, args)
	}
}
