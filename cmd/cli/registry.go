// ork registry — push, pull, info, list
//
// All four commands follow the same pattern as ork notes and ork init:
// minimal flags, clear output, no hidden state.

//go:build !runtime && !gateway

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

  export ORK_REGISTRY=oci://myregistry.internal/patterns
  export ORK_MOTIFS_REGISTRY=oci://myregistry.internal/motifs`,
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
	registryPullCmd.Flags().StringP("file", "f", "", "Pull all OCI imports from a katalog or komposer file")

	registryListCmd.Flags().StringP("tag", "t", "", "Filter by tag (e.g. database, stateful, security)")
	registryListCmd.Flags().BoolP("katalogs", "k", false, "Show only katalogs (kind: Katalog)")
	registryListCmd.Flags().BoolP("motifs", "m", false, "Show only motifs (kind: Motif)")
	registryPushCmd.Flags().BoolVar(&registryPushForce, "force", false, "force push even if metadata.version differs from tag or e2e fails")
	registryPushCmd.Flags().BoolVar(&registryPushUpdateMeta, "update-meta", false, "persist overridden metadata.version back to the primary file")
	registryPushCmd.Flags().StringVar(&registryPushE2EFile, "e2e", "", "path to e2e spec file (default: e2e.yaml in pattern dir)")
	registryPushCmd.Flags().BoolVar(&registryPushNoE2E, "no-e2e", false, "skip the e2e gate even if e2e.yaml is present")
	registryPushCmd.Flags().BoolVar(&registryPushNoSimulate, "no-simulate", false, "skip the simulate gate even if simulate.yaml is present")

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
