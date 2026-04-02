package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/utils"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v2"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Render and print the merged Katalog CRDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		var k katalog.Katalog
		if _, err = k.KomposeKatalogFromYaml(m.m); err != nil {
			return err
		}
		if _, err = k.ValidateConfig(kfg); err != nil {
			return err
		}

		crds := k.Enabled()

		verbose, _ := cmd.Flags().GetBool("verbose")
		jsonOut, _ := cmd.Flags().GetBool("json")
		yamlOut, _ := cmd.Flags().GetBool("yaml")
		graphOut, _ := cmd.Flags().GetBool("graph")

		switch {
		case jsonOut:
			dtos := make([]CRDEntryDTO, 0, len(crds))
			for _, crd := range crds {
				dtos = append(dtos, toDTO(crd))
			}
			b, _ := json.MarshalIndent(dtos, "", "  ")
			fmt.Println(string(b))

		case yamlOut:
			dtos := make([]CRDEntryDTO, 0, len(crds))
			for _, crd := range crds {
				dtos = append(dtos, toDTO(crd))
			}
			b, _ := yaml.Marshal(dtos)
			fmt.Println(string(b))

		case graphOut:
			printGraph(crds)

		case verbose:
			fmt.Println("Verbose merged katalog output:")
			for _, crd := range crds {
				printPrettyCRD(crd)
			}

		default:
			fmt.Printf("Success: %sKatalog is valid%s\n\n", utils.ColorGreen, utils.ColorReset)
			fmt.Println("Rendered CRDs:")
			for _, crd := range crds {
				fmt.Printf("  - %s", crd.Name)
				if len(crd.DependsOn) > 0 {
					fmt.Printf("  (depends on: %v)", strings.Join(crd.DependsOn, ", "))
				}
				fmt.Println()
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.Flags().BoolP("verbose", "v", false, "Show full merged katalog output")
	templateCmd.Flags().BoolP("json", "j", false, "Output CRDs as JSON")
	templateCmd.Flags().BoolP("yaml", "y", false, "Output CRDs as YAML")
	templateCmd.Flags().BoolP("graph", "g", false, "Show ASCII dependency graph")
}
