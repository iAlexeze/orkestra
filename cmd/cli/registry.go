// cmd/cli/registry.go
//
// ork registry — push, pull, info, list
//
// All four commands follow the same pattern as ork notes and ork init:
// minimal flags, clear output, no hidden state.

//go:build !runtime

package cli

import "github.com/spf13/cobra"

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Push, pull, and inspect Orkestra patterns from OCI registries",
	Long: `Manage Orkestra patterns (patterns and motifs) in OCI registries.

  ork registry push <name>:<version> <dir>    push a pattern or motif directory
  ork registry pull <name>:<version>          pull an pattern to local cache
  ork registry info <name>:<version>          show pattern metadata
  ork registry list [registry-url]            list available patterns

Authentication uses ~/.docker/config.json — run 'docker login' first.
Override the default registries with environment variables:

  export ORKESTRA_REGISTRY=oci://myregistry.internal/patterns
  export ORKESTRA_MOTIFS_REGISTRY=oci://myregistry.internal/motifs`,
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
	registryListCmd.Flags().BoolP("katalogs", "k", false, "Show only katalogs (kind: Katalog)")
	registryListCmd.Flags().BoolP("motifs", "m", false, "Show only motifs (kind: Motif)")
	registryPushCmd.Flags().BoolVar(&registryPushForce, "force", false, "force push even if metadata.version differs from tag")
	registryPushCmd.Flags().BoolVar(&registryPushUpdateMeta, "update-meta", false, "persist overridden metadata.version back to the primary file")

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
