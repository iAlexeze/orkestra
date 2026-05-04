//go:build !runtime

package cli

import (
	"fmt"
	"os"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var komposeCmd = &cobra.Command{
	Use:   "kompose",
	Short: "Resolve a Komposer into a single merged Katalog",
	Long: `Reads a komposer.yaml, validates it is kind: Komposer, resolves all sources,
and emits a fully merged Katalog suitable for use with Orkestra runtime.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outFile, _ := cmd.Flags().GetString("output")
		katalogPaths, _ := cmd.Flags().GetStringSlice("katalog")

		if len(katalogPaths) != 1 {
			return fmt.Errorf("kompose expects exactly one --katalog file")
		}
		komposerPath := katalogPaths[0]

		// Step 1: ensure the input is a Komposer
		data, err := os.ReadFile(komposerPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", komposerPath, err)
		}

		doc, err := merger.ParseKatalogDoc(data, komposerPath)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", komposerPath, err)
		}

		if doc == nil {
			return fmt.Errorf("document is not valid")
		}

		if doc.Kind != konfig.KomposerKind() {
			return fmt.Errorf("expected kind: Komposer, got: %s", doc.Kind)
		}

		// Step 2: merge using generateKatalog machinery (this resolves all sources)
		out, err := generateKatalog(cmd)
		if err != nil {
			return fmt.Errorf("merge komposer: %w", err)
		}

		kat := out.kat
		kat.APIVersion = doc.APIVersion
		kat.Kind = konfig.KatalogKind()
		kat.KomposerMetadata = doc.Metadata

		// Populate all top-level fields from the merged result — the merger
		// accumulates security, notification, and providers from every source
		// Katalog additively, then lets the Komposer's own declarations win.
		kat.Security = out.m.ToSecurity()
		kat.Notification = out.m.ToNotification()
		kat.Providers = out.m.ToProviders()

		// Step 3: validate merged katalog
		validKat, err := out.kat.ValidateConfig(kfg)
		if err != nil {
			return fmt.Errorf("validate merged katalog: %w", err)
		}

		kat.Spec = validKat.Spec

		// Step 4: render merged katalog YAML
		rendered, err := yaml.Marshal(kat)
		if err != nil {
			return fmt.Errorf("marshal merged katalog: %w", err)
		}

		// Step 5: prune empty fields and write to file or stdout
		pruned, err := pruneEmptyYAML(rendered)
		if err != nil {
			return err
		}

		// Write pruned output
		if outFile != "" {
			if err := os.WriteFile(outFile, pruned, 0644); err != nil {
				return fmt.Errorf("writing %s: %w", outFile, err)
			}
			fmt.Printf("merged katalog written to %s\n", outFile)
		} else {
			fmt.Println(string(pruned))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(komposeCmd)

	komposeCmd.Flags().StringP("output", "o", "", "Write merged katalog to file")
	komposeCmd.Flags().StringSliceP("katalog", "k", nil, "Path to komposer.yaml")

	// Shadow global flags so they don't appear under `ork kompose`
	komposeCmd.Flags().Bool("debug", false, "")
	komposeCmd.Flags().String("kubeconfig", "", "")
	komposeCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	komposeCmd.Flags().MarkHidden("debug")
	komposeCmd.Flags().MarkHidden("kubeconfig")
	komposeCmd.Flags().MarkHidden("verbose")
}
