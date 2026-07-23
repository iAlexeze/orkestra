//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/tools/generate"
	"github.com/spf13/cobra"
)

var createPatternCmd = &cobra.Command{
	Use:   "pattern",
	Short: "Scaffold a new Orkestra pattern: katalog.yaml, simulate.yaml, e2e.yaml, README.md",
	Long: `Creates the files needed to build, test, and publish an Orkestra pattern.

Always written:
  katalog.yaml   — operator declaration
  simulate.yaml  — in-memory test scaffold (ork simulate)
  e2e.yaml       — real-cluster integration test scaffold (ork e2e)
  README.md      — actionable steps from edit to release

Also written when --typed, --add-hook, or --add-constructor:
  values.yaml    — runtime image (set before ork e2e)
  Makefile       — registry, build, build-runtime, docker, push, release
  Dockerfile     — production container image (distroless, runtime binary only)

Typed mode flags are forwarded to katalog generation:
  --add-hook          Include a hooks section
  --add-constructor   Include a constructor section
  --typed             Include both hooks and constructor (commented)

Examples:
  ork create pattern
  ork create pattern --add-hook -o ./my-operator/
  ork create pattern --typed`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addHook, _ := cmd.Flags().GetBool("add-hook")
		addConstructor, _ := cmd.Flags().GetBool("add-constructor")
		typed, _ := cmd.Flags().GetBool("typed")
		outputDir, _ := cmd.Flags().GetString("output")

		if outputDir == "" {
			outputDir = "."
		}

		isTyped := typed || addHook || addConstructor

		katalogOpts := generate.KatalogScaffoldOptions{
			AddHook:        addHook,
			AddConstructor: addConstructor,
			Typed:          typed,
			OutputFile:     filepath.Join(outputDir, fileKatalog),
		}
		if err := katalogOpts.Validate(); err != nil {
			return err
		}

		fmt.Printf("generating pattern scaffold → %s/\n", outputDir)

		if _, err := generate.KatalogScaffold(katalogOpts); err != nil {
			return fmt.Errorf("generating %s: %w", fileKatalog, err)
		}

		if err := generate.WriteSimulateScaffold(filepath.Join(outputDir, fileSimulate)); err != nil {
			return fmt.Errorf("generating %s: %w", fileSimulate, err)
		}

		if err := generate.WriteE2EScaffold(filepath.Join(outputDir, fileE2e), isTyped); err != nil {
			return fmt.Errorf("generating %s: %w", fileE2e, err)
		}

		if err := generate.WriteREADME(filepath.Join(outputDir, fileReadMe), isTyped); err != nil {
			return fmt.Errorf("generating %s: %w", fileReadMe, err)
		}

		if isTyped {
			if err := generate.WriteValuesYAML(filepath.Join(outputDir, fileValues)); err != nil {
				return fmt.Errorf("generating %s: %w", fileValues, err)
			}
			if err := generate.WriteMakefile(filepath.Join(outputDir, fileMakeFile)); err != nil {
				return fmt.Errorf("generating %s: %w", fileMakeFile, err)
			}
			if err := generate.WriteDockerfile(filepath.Join(outputDir, fileDockerfile)); err != nil {
				return fmt.Errorf("generating %s: %w", fileDockerfile, err)
			}
		}

		fmt.Printf("\n→ pattern scaffold written to %s\n", bold(outputDir+"/"))
		fmt.Printf("  %s %-16s %s\n", successMark(), fileKatalog, dim("declare your CRD(s) and resources"))
		fmt.Printf("  %s %-16s %s\n", successMark(), fileSimulate, dim("ork simulate"))
		fmt.Printf("  %s %-16s %s\n", successMark(), fileE2e, dim("ork e2e"))
		fmt.Printf("  %s %-16s %s\n", successMark(), fileReadMe, dim("start here"))
		if isTyped {
			fmt.Printf("  %s %-16s %s\n", successMark(), fileValues, dim("set runtime.image before ork e2e"))
			fmt.Printf("  %s %-16s %s\n", successMark(), fileMakeFile, dim("make registry, make build, make release"))
			fmt.Printf("  %s %-16s %s\n", successMark(), fileDockerfile, dim("production container image"))
		}
		fmt.Println()
		return nil
	},
}

func init() {
	createCmd.AddCommand(createPatternCmd)
	createPatternCmd.Flags().Bool("add-hook", false,
		"Typed mode: include a hooks section in katalog.yaml (also writes Makefile + Dockerfile)")
	createPatternCmd.Flags().Bool("add-constructor", false,
		"Typed mode: include a constructor section in katalog.yaml (also writes Makefile + Dockerfile)")
	createPatternCmd.Flags().Bool("typed", false,
		"Typed mode: include both hooks and constructor commented (also writes Makefile + Dockerfile)")
	createPatternCmd.Flags().StringP("output", "o", "",
		"Output directory (default: current directory)")
}
