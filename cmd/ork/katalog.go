package ork

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/spf13/cobra"
)

var katalogCmd = &cobra.Command{
	Use:   "katalog",
	Short: "Interact with the Orkestra CRD katalog",
}

var katalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List CRDs in the katalog",
	RunE: func(cmd *cobra.Command, args []string) error {

		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		fmt.Println("CRDs in katalog:")
		for _, crd := range k.All() {
			status := "disabled"
			if crd.Enabled {
				status = "enabled"
			}
			fmt.Printf("- %-20s (%s)\n", crd.Name, status)
		}

		return nil
	},
}

var katalogDescribeCmd = &cobra.Command{
	Use:   "describe <crd>",
	Short: "Describe a CRD in the katalog",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		out, err := k.Describe(name)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var explainCmd = &cobra.Command{
	Use:   "explain <crd>",
	Short: "Explain how Orkestra handles a CRD",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		out, err := k.Explain(name)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var getCRDsCmd = &cobra.Command{
	Use:   "crds",
	Short: "List enabled CRDs",
	RunE: func(cmd *cobra.Command, args []string) error {

		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		fmt.Println("Enabled CRDs:")
		for _, crd := range k.Enabled() {
			fmt.Printf("- %s (%s/%s)\n", crd.Name, crd.Group, crd.Version)
		}

		return nil
	},
}

var getControllersCmd = &cobra.Command{
	Use:   "controllers",
	Short: "List active controllers",
	RunE: func(cmd *cobra.Command, args []string) error {

		k := katalog.NewKatalog(kfg.Katalog().Mode, kfg.Katalog().Path)

		fmt.Println("Controllers:")
		for _, crd := range k.Enabled() {
			if crd.ReconcilerConfig.Default && crd.ReconcilerConfig.Constructor != nil {
				fmt.Printf("- %s\n", crd.Name)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(katalogCmd)
	katalogCmd.AddCommand(katalogListCmd)
	katalogCmd.AddCommand(katalogDescribeCmd)
	katalogCmd.AddCommand(explainCmd)
	katalogCmd.AddCommand(getCRDsCmd)
	katalogCmd.AddCommand(getControllersCmd)
}
