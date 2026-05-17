//go:build !runtime

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// templateRuntimeOutput is the serializable view of a fully-expanded Katalog.
// It holds what the runtime receives after motif expansion and validation.
type templateRuntimeOutput struct {
	APIVersion string             `yaml:"apiVersion" json:"apiVersion"`
	Kind       string             `yaml:"kind"       json:"kind"`
	Metadata   map[string]string  `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Spec       templateSpecOutput `yaml:"spec"       json:"spec"`
}

type templateSpecOutput struct {
	CRDs map[string]orktypes.CRDEntry `yaml:"crds" json:"crds"`
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Render and inspect the fully-expanded runtime Katalog",
	Long: `Loads one or more Katalog or Komposer files, fully expands all motif imports,
resolves all template inputs, validates the configuration, and shows exactly
what the runtime will see — nothing more, nothing less.

By default prints a human-readable summary. Use flags to switch output format
or drill into a specific CRD.

Examples:
  ork template -f katalog.yaml
  ork template -f katalog.yaml --yaml
  ork template -f katalog.yaml --json
  ork template -f katalog.yaml --graph
  ork template -f katalog.yaml --crd pipeline
  ork template -f katalog.yaml --yaml -o runtime.yaml
  ork template -f a.yaml -f b.yaml --graph`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		yamlOut, _ := cmd.Flags().GetBool("yaml")
		graphOut, _ := cmd.Flags().GetBool("graph")
		crdName, _ := cmd.Flags().GetString("crd")
		outFile, _ := cmd.Flags().GetString("output")
		noValidate, _ := cmd.Flags().GetBool("no-validate")

		// ── Load & expand ───────────────────────────────────────────────────────
		merged, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		var k katalog.Katalog
		if _, err = k.KomposeRuntimeKatalog(kfg, merged.m); err != nil {
			return err
		}

		if !noValidate {
			if _, err = k.ValidateConfig(kfg); err != nil {
				return fmt.Errorf("validation: %w", err)
			}
		}

		crds := k.Enabled()
		depGraph := katalog.NewDependencyGraph(&k)
		startupOrder := depGraph.StartupOrder()

		// ── Route to output mode ────────────────────────────────────────────────
		var out []byte

		switch {

		// ── --crd <name>: drill into one CRD ───────────────────────────────────
		case crdName != "":
			crd, ok := crds[crdName]
			if !ok {
				available := make([]string, 0, len(crds))
				for n := range crds {
					available = append(available, n)
				}
				sort.Strings(available)
				return fmt.Errorf("CRD %q not found — available: %v", crdName, available)
			}
			if jsonOut {
				out, err = json.MarshalIndent(crd, "", "  ")
			} else if yamlOut {
				out, err = yaml.Marshal(crd)
			} else {
				printCRDDetail(crd, depGraph)
				return nil
			}

		// ── --graph: dependency tree ────────────────────────────────────────────
		case graphOut:
			printDependencyGraph(crds, depGraph, startupOrder)
			return nil

		// ── --yaml / --json: full runtime Katalog ───────────────────────────────
		case yamlOut || jsonOut:
			view := buildRuntimeView(&k)
			if jsonOut {
				out, err = json.MarshalIndent(view, "", "  ")
			} else {
				raw, merr := yaml.Marshal(view)
				if merr != nil {
					return fmt.Errorf("marshal: %w", merr)
				}
				pruned, perr := pruneEmptyYAML(raw)
				if perr != nil {
					return fmt.Errorf("prune: %w", perr)
				}
				out = append([]byte("---\n"), pruned...)
			}

		// ── default: human-readable summary ────────────────────────────────────
		default:
			printTemplateSummary(&k, crds, startupOrder)
			return nil
		}

		if err != nil {
			return fmt.Errorf("render: %w", err)
		}

		if outFile != "" {
			if yamlOut {
				if err := utils.WriteFileAndFormat(outFile, out, 0644); err != nil {
					return fmt.Errorf("writing %s: %w", outFile, err)
				}
			} else {
				if err := os.WriteFile(outFile, out, 0644); err != nil {
					return fmt.Errorf("writing %s: %w", outFile, err)
				}
			}
			fmt.Printf("%s\n", utils.Green("written to "+outFile))
			return nil
		}
		fmt.Println(string(out))
		return nil
	},
}

// buildRuntimeView constructs the serializable runtime view from the expanded Katalog.
func buildRuntimeView(k *katalog.Katalog) templateRuntimeOutput {
	meta := k.Metadata()
	metaFields := map[string]string{}
	if meta.Name != "" {
		metaFields["name"] = meta.Name
	}
	if meta.Version != "" {
		metaFields["version"] = meta.Version
	}
	if meta.Author != "" {
		metaFields["author"] = meta.Author
	}
	if meta.Description != "" {
		metaFields["description"] = meta.Description
	}
	if len(metaFields) == 0 {
		metaFields = nil
	}

	return templateRuntimeOutput{
		APIVersion: k.APIVersion,
		Kind:       k.Kind,
		Metadata:   metaFields,
		Spec: templateSpecOutput{
			CRDs: k.Enabled(),
		},
	}
}

func init() {
	rootCmd.AddCommand(templateCmd)

	templateCmd.Flags().BoolP("yaml", "y", false, "Output full expanded runtime Katalog as YAML")
	templateCmd.Flags().BoolP("json", "j", false, "Output full expanded runtime Katalog as JSON")
	templateCmd.Flags().BoolP("graph", "g", false, "Show dependency graph with startup order")
	templateCmd.Flags().StringP("crd", "c", "", "Drill into a specific CRD by name")
	templateCmd.Flags().StringP("output", "o", "", "Write output to file instead of stdout")
	templateCmd.Flags().Bool("no-validate", false, "Skip validation (show expanded state even if invalid)")

	// Shadow global flags
	templateCmd.Flags().Bool("debug", false, "")
	templateCmd.Flags().String("kubeconfig", "", "")
	templateCmd.Flags().Bool("verbose", false, "")

	templateCmd.Flags().MarkHidden("debug")
	templateCmd.Flags().MarkHidden("kubeconfig")
	templateCmd.Flags().MarkHidden("verbose")
}
