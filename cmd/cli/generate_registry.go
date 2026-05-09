//go:build !runtime

package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/generate"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/spf13/cobra"
)

// Generate 'pkg/runtime/zz_generated_runtime_registry.go' for one or more operator projects.
//
// This command scans a Katalog, resolves all CRDs, typed extensions, and
// managed‑resource contracts, and produces a complete runtime registry:
//
//   - CRD registrations
//   - Reconciler registrations (hooks, constructors, operatorBox)
//   - List/watch registrations
//   - Runtime object wiring
//
// The registry is written to:
//
//	pkg/runtime/zz_generated_runtime_registry.go
//
// A matching cmd/orkestra/main.go file is also ensured, containing the required
// blank import for the runtime package.
//
// The command supports both single‑project and multi‑project generation:
//
//   - Single project: run in the project directory (must contain go.mod)
//   - Multi‑project: use --dirs to generate registries for multiple operator
//     projects in one invocation
//
// Examples:
//
//	# Generate registry for the current project
//	ork generate registry --file katalog.yaml
//
//	# Generate registry for a specific katalog file
//	ork generate registry --file path/to/katalog.yaml
//
//	# Generate registry for multiple operator projects
//	ork generate registry --dirs ./website,./database,./pipeline
//
//	# Dry‑run (print generated files without writing them)
//	ork generate registry --file katalog.yaml --dry-run
//
// Each project directory must contain:
//   - go.mod
//   - a katalog.yaml (or --file pointing to one)
//   - pkg/runtime/ (created automatically if missing)
//
// The generated registry is deterministic and reflects the exact typed‑mode
// contracts declared in the Katalog (hooks, constructors, operatorBox).
var generateRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Generate 'pkg/runtime/zz_generated_runtime_registry.go' from a Katalog",
	Long: `Generate the Orkestra runtime registry for one or multiple operator projects.

This command scans a Katalog, resolves all CRDs, typed extensions, and
managed‑resource contracts, and produces the runtime registry used by the
operator process. The registry includes:

  • CRD registrations
  • Reconciler registrations (hooks, constructors, operatorBox)
  • List/watch registrations
  • Runtime object wiring

The registry is written to:
  pkg/runtime/zz_generated_runtime_registry.go

A matching cmd/orkestra/main.go file is also ensured, containing the required
blank import for the runtime package.

You can generate the registry for the current project, or for multiple operator
projects in one invocation using --dirs.

Examples:
  # Generate registry for the current project
  ork generate registry --file katalog.yaml

  # Generate registry for a specific katalog file
  ork generate registry --file path/to/katalog.yaml

  # Generate registries for multiple operator projects
  ork generate registry --dirs ./database,./pipeline --file komposer.yaml

  # Dry‑run (print generated files without writing them)
  ork generate registry --file katalog.yaml --dry-run

Each project directory must contain:
  • go.mod
  • a katalog.yaml
  • pkg/runtime/ (created automatically if missing)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		dirs, _ := cmd.Flags().GetString("dirs")

		// Multi-directory mode
		if dirs != "" {
			for _, d := range strings.Split(dirs, ",") {
				abs := filepath.Clean(d)
				if err := generateRegistryForDir(abs, cmd, dryRun); err != nil {
					return fmt.Errorf("registry generation failed for %s: %w", abs, err)
				}
			}
			return nil
		}

		// Single-directory mode (current working directory)
		cwd, _ := os.Getwd()
		return generateRegistryForDir(cwd, cmd, dryRun)
	},
}

// generateRegistryForDir performs the full registry generation pipeline
// for a single operator project directory.
func generateRegistryForDir(dir string, cmd *cobra.Command, dryRun bool) error {
	logger.Info().Str("dir", dir).Msg("generating registry for project")

	// Save original working directory
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	// Enter target project directory
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("chdir %s: %w", dir, err)
	}

	// Load katalog (from --file or default)
	out, err := generateKatalog(cmd)
	if err != nil {
		return fmt.Errorf("loading katalog: %w", err)
	}

	// Validate Go project
	root, moduleName, err := validateProject()
	if err != nil {
		return fmt.Errorf("validating project: %w", err)
	}

	// Generate runtime registry
	logger.Info().Msg("generating runtime registry...")
	if err := generate.Runtime(out.m, dryRun); err != nil {
		return fmt.Errorf("generate runtime registry: %w", err)
	}

	logger.Info().
		Str("registry", filepath.Join(generate.RuntimePackage, generate.RegistryFile)).
		Msg("runtime registry generated successfully")

	// Ensure main.go exists
	if err := ensureMainGo(root, moduleName, dryRun); err != nil {
		logger.Warn().Err(err).Msg("failed to ensure main.go – you may need to add import manually")
	}

	return nil
}

// validateProject verifies the current working directory is inside a Go module.
func validateProject() (string, string, error) {
	root, err := findProjectRoot()
	if err != nil {
		return "", "", fmt.Errorf("finding project root: %w", err)
	}

	moduleName, err := readModuleName(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", "", fmt.Errorf("reading module name: %w", err)
	}

	logger.Debug().
		Str("project_root", root).
		Str("module", moduleName).
		Msg("validated Go project")

	return root, moduleName, nil
}

// ensureMainGo writes cmd/orkestra/main.go with the required blank import.
func ensureMainGo(root, moduleName string, dryRun bool) error {
	targetDir := filepath.Join(root, "cmd", "orkestra")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating cmd/orkestra: %w", err)
	}
	mainPath := filepath.Join(targetDir, "main.go")

	timestamp := time.Now().UTC().Format(time.RFC3339)
	importLine := fmt.Sprintf(`_ "%s/pkg/runtime"`, moduleName)

	content := fmt.Sprintf(`// Code generated by "ork generate registry" on %s. DO NOT EDIT.
// Re-generate by running: ork generate registry --file <path-or-url>
package main

import (
    "context"

    "github.com/orkspace/orkestra/cmd/cli"
    "github.com/orkspace/orkestra/pkg/konfig"
    "github.com/orkspace/orkestra/pkg/logger"
    "github.com/orkspace/orkestra/pkg/utils"

    %s
)

func main() {
    kfg, err := konfig.Init()
    if err != nil {
        logger.Fatal().AnErr("failed to load configurations", err)
        utils.Exit(err)
    }
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    cli.Execute(kfg, ctx)
}`, timestamp, importLine)

	if dryRun {
		fmt.Println(content)
		return nil
	}

	return os.WriteFile(mainPath, []byte(content), 0644)
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir, nil
	}
	return "", fmt.Errorf("go.mod not found in current directory: %s", dir)
}

func readModuleName(modPath string) (string, error) {
	f, err := os.Open(modPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("module declaration not found in %s", modPath)
}
