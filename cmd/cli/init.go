//go:build !runtime && !gateway

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
	Use:   "init [project-name]",
	Short: "Initialize a new Orkestra operator project",
	Args: func(cmd *cobra.Command, args []string) error {
		list, _ := cmd.Flags().GetBool("list-packs")
		clear, _ := cmd.Flags().GetBool("clear-cache")

		if list || clear {
			return nil
		}

		if len(args) > 1 {
			return fmt.Errorf("too many arguments — usage: ork init [project-name]")
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

		// No name → init in current directory, like terraform init
		name := "."
		if len(args) == 1 {
			name = args[0]
		}

		pack, _ := cmd.Flags().GetString("pack")
		refresh, _ := cmd.Flags().GetBool("refresh-cache")

		if pack == "" {
			return initCanonical(name)
		}
		return initProject(name, pack, refresh)
	},
}

func initCanonical(name string) error {
	printBanner()
	label := nameLabel(name)
	fmt.Printf("Initialising %s...\n\n", utils.Bold(label))

	steps := []initStep{}
	if name != "." {
		steps = append(steps, initStep{"Creating project folder", func() error { return os.MkdirAll(name, 0755) }})
	}
	steps = append(steps, initStep{"Writing katalog.yaml", func() error { return extractCanonical(name) }})

	if err := runSteps(steps); err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", utils.Green("✅ Project ready: "+label))
	if name != "." {
		fmt.Printf("  cd %s\n", name)
	}
	fmt.Printf("  ork run\n\n")
	fmt.Println("To explore example packs:")
	fmt.Printf("  ork init %s --pack beginner\n", label)

	return nil
}

// nameLabel returns the display name for a project path.
// When initialising in the current directory, it uses the directory's base name.
func nameLabel(name string) string {
	if name != "." {
		return name
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Base(cwd)
}

func initProject(name, pack string, refresh bool) error {

	printBanner()
	fmt.Printf("Initialising %s using '%s' example pack...\n\n",
		utils.Bold(name), pack)

	label := nameLabel(name)
	ver := version.Version // ldflags

	steps := []initStep{}
	if name != "." {
		steps = append(steps, initStep{"Creating project folder", func() error { return os.MkdirAll(name, 0755) }})
	}

	if refresh {
		steps = append(steps,
			initStep{"Downloading example pack", func() error { return downloadExamplePack(name, pack, ver, refresh) }},
			initStep{"Extracting example pack", func() error { return extractExamplePack(name, pack, ver) }},
		)
	} else {
		steps = append(steps,
			initStep{"Extracting example pack", func() error { return extractEmbeddedPack(name, pack) }},
		)
	}

	if err := runSteps(steps); err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", utils.Green("✅ Project ready: "+label))
	if name != "." {
		fmt.Printf("  cd %s\n", name)
	}
	fmt.Printf("  ls %s/\n\n", pack)
	fmt.Println("To run an example:")
	if name != "." {
		fmt.Printf("  cd %s/%s/01-hello-website\n", name, pack)
	} else {
		fmt.Printf("  cd %s/01-hello-website\n", pack)
	}
	fmt.Printf("  ork run\n\n")
	fmt.Println("Control Center:")
	fmt.Printf("  ork control    # opens localhost:8081\n\n")

	return nil
}

func listPacks() error {
	printBanner()

	fmt.Printf("Available example packs:\n")

	for _, p := range ListPacks() {
		fmt.Printf("  %-13s → %s\n", p.Name, p.Description)
	}

	fmt.Printf("\nNo pack — canonical hello-website operator:\n")
	fmt.Printf("  ork init              # init in current directory\n")
	fmt.Printf("  ork init my-operator  # init in new folder\n\n")
	fmt.Printf("With a pack:\n")
	fmt.Printf("  ork init --pack <name>\n")
	fmt.Printf("  ork init my-operator --pack <name>\n")

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("pack", "p", "", "Example pack to copy into the project (beginner, intermediate, advanced, …)")
	initCmd.Flags().BoolP("list-packs", "l", false, "List available example packs")
	initCmd.Flags().Bool("clear-cache", false, "Clear cached example packs")
	initCmd.Flags().Bool("refresh-cache", false, "Fetch pack from GitHub Releases instead of using the built-in copy")

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
