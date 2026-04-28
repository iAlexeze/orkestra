//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/orkspace/orkestra/pkg/version"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Initialize a new Orkestra operator project",
	Args: func(cmd *cobra.Command, args []string) error {
		list, _ := cmd.Flags().GetBool("list-packs")
		clear, _ := cmd.Flags().GetBool("clear-cache")
		refresh, _ := cmd.Flags().GetBool("refresh-cache")

		// Utility flags do NOT require a project name
		if list || clear || refresh {
			return nil
		}

		// Normal init requires exactly 1 argument
		if len(args) != 1 {
			return fmt.Errorf("project name is required. Example: ork init my-operator")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --list-packs
		list, _ := cmd.Flags().GetBool("list-packs")
		if list {
			return listPacks()
		}

		// Handle --clear-cache
		clear, _ := cmd.Flags().GetBool("clear-cache")
		if clear {
			return clearCache()
		}

		// Normal init flow
		name := args[0]
		pack, _ := cmd.Flags().GetString("pack")
		if pack == "" {
			pack = "beginner"
		}

		refresh, _ := cmd.Flags().GetBool("refresh-cache")

		return initProject(name, pack, refresh)
	},
}

func initProject(name, pack string, refresh bool) error {

	printBanner()
	fmt.Printf("Initialising %s%s%s using '%s' example pack...\n\n",
		utils.ColorBold, name, utils.ColorReset, pack)

	version := version.Version // ldflags

	steps := []initStep{
		{"Creating project folder", func() error { return os.MkdirAll(name, 0755) }},
		{"Creating examples folder", func() error { return os.MkdirAll(filepath.Join(name, "examples"), 0755) }},
		{"Downloading example pack", func() error { return downloadExamplePack(name, pack, version, refresh) }},
		{"Extracting example pack", func() error { return extractExamplePack(name, pack, version) }},
	}

	if err := runSteps(steps); err != nil {
		return err
	}

	fmt.Printf("\n%s✅ Project ready: %s%s\n\n", utils.ColorGreen, name, utils.ColorReset)
	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  kubectl apply -f examples/%s/01-hello-website/crd.yaml\n", pack)
	fmt.Printf("  ork run --katalog examples/%s/01-hello-website/katalog.yaml\n", pack)
	fmt.Println()
	fmt.Println("Apply the CR:")
	fmt.Printf("  kubectl apply -f examples/%s/01-hello-website/cr.yaml\n", pack)
	fmt.Println()
	fmt.Println("View on Control Center → localhost:8090")
	fmt.Printf("  ork control start\n\n")

	return nil
}

func listPacks() error {
	printBanner()

	fmt.Printf("Available example packs:\n")

	packs := map[string]string{
		"beginner":     "Start here. Simple CRDs, Deployments, Services.",
		"intermediate": "Multi-resource patterns, when/anyOf, Komposer basics.",
		"advanced":     "Hooks, constructors, validation/mutation, registries.",
		"use-cases":    "Full-stack, cross-CRD, external gates, once-secrets.",
		"rollback":     "Zero-config and configurable failure recovery",
	}

	for name, desc := range packs {
		fmt.Printf("  %-13s → %s\n", name, desc)
	}

	fmt.Printf("\nUse:")
	fmt.Printf("  ork init <project-name> --pack <name>\n")
	fmt.Println("Default:")
	fmt.Printf("  ork init <project-name>        # uses beginner\n")

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("pack", "p", "", "Example pack to initialize (default: beginner)")
	initCmd.Flags().BoolP("list-packs", "l", false, "List available example packs")
	initCmd.Flags().Bool("clear-cache", false, "Clear cached example packs")
	initCmd.Flags().Bool("refresh-cache", false, "Force re-download of example pack")

	// Shadow global flags so they don't appear under `ork init`
	initCmd.Flags().Bool("debug", false, "")
	initCmd.Flags().String("kubeconfig", "", "")
	initCmd.Flags().StringSlice("katalog", nil, "")
	initCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	initCmd.Flags().MarkHidden("debug")
	initCmd.Flags().MarkHidden("kubeconfig")
	initCmd.Flags().MarkHidden("katalog")
	initCmd.Flags().MarkHidden("verbose")
}
