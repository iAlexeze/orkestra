// cmd/cli/init.go
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ialexeze/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Scaffold a new Orkestra operator project",
	Long: `Scaffold a new Orkestra operator project.

By default, creates a dynamic project — Katalog files only.
No Go code, no compilation. Run with:

  ork run --katalog katalog.yaml

For operators that need Go code (typed CRDs, custom reconcilers):

  ork init my-operator --typed

This creates a full Go project that you build and run as your own binary.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		typed, _ := cmd.Flags().GetBool("typed")
		module, _ := cmd.Flags().GetString("module")

		if typed && module == "" {
			module = "github.com/myorg/" + name
		}

		if typed {
			return initTyped(name, module)
		}
		return initDynamic(name)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("typed", false, "Scaffold a Go project for typed CRDs or custom reconcilers")
	initCmd.Flags().String("module", "", "Go module path — only used with --typed (default: github.com/myorg/<name>)")
}

// ── Dynamic init — no Go required ─────────────────────────────────────────────
// The ork binary IS the operator. No compilation needed.
// The user writes Katalog YAML and runs ork run directly.

func initDynamic(name string) error {
	printBanner()
	fmt.Printf("Initialising %s%s%s (dynamic — zero Go required)...\n\n", utils.ColorBold, name, utils.ColorReset)

	steps := []initStep{
		{"Creating folder structure", func() error { return createDynamicFolders(name) }},
		{"Writing website example Katalog", func() error { return writeExampleKatalog(name) }},
		{"Writing website CRD", func() error { return writeExampleCRD(name) }},
		{"Writing sample CR", func() error { return writeExampleCR(name) }},
		{"Writing .env.example", func() error { return writeEnvExample(name) }},
		{"Writing .gitignore", func() error { return writeDynamicGitignore(name) }},
		{"Writing README", func() error { return writeDynamicReadme(name) }},
	}

	if err := runSteps(steps); err != nil {
		return err
	}

	fmt.Printf("\n%s✅ Project ready: %s%s\n\n", utils.ColorGreen, name, utils.ColorReset)
	fmt.Println("No Go required. Start your operator:")
	fmt.Println()
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  kubectl apply -f examples/website/website-crd.yaml\n")
	fmt.Printf("  ork run --katalog examples/website/website-katalog.yaml\n")
	fmt.Println()
	fmt.Printf("Validate first:\n")
	fmt.Printf("  ork validate --katalog examples/website/website-katalog.yaml\n")
	fmt.Println()

	return nil
}

// ── Typed init — Go project ────────────────────────────────────────────────────
// Used when the operator needs compiled Go types, Go hooks, or custom reconcilers.
// The user builds their own binary from this scaffold.

func initTyped(name, module string) error {
	printBanner()
	fmt.Printf("Initialising %s%s%s (typed — Go project)...\n\n", utils.ColorBold, name, utils.ColorReset)

	steps := []initStep{
		{"Creating folder structure", func() error { return createTypedFolders(name) }},
		{"Writing main.go", func() error { return writeMain(name) }},
		{"Writing go.mod", func() error { return writeGoMod(name, module) }},
		{"Writing example Katalog", func() error { return writeExampleKatalog(name) }},
		{"Writing example CRD", func() error { return writeExampleCRD(name) }},
		{"Writing sample CR", func() error { return writeExampleCR(name) }},
		{"Writing .env.example", func() error { return writeEnvExample(name) }},
		{"Writing .gitignore", func() error { return writeTypedGitignore(name) }},
		{"Writing README", func() error { return writeTypedReadme(name, module) }},
		{"Downloading dependencies (go mod tidy)", func() error { return goModTidy(name) }},
	}

	if err := runSteps(steps); err != nil {
		return err
	}

	fmt.Printf("\n%s✅ Project ready: %s%s\n\n", utils.ColorGreen, name, utils.ColorReset)
	fmt.Println("Build and run your operator:")
	fmt.Println()
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  kubectl apply -f examples/website/website-crd.yaml\n")
	fmt.Printf("  go run ./cmd/orkestra/ run --katalog examples/website/website-katalog.yaml\n")
	fmt.Println()
	fmt.Println("Or build a binary:")
	fmt.Printf("  go build -o ./bin/%s ./cmd/orkestra/\n", name)
	fmt.Printf("  ./bin/%s run --katalog examples/website/website-katalog.yaml\n", name)
	fmt.Println()

	return nil
}

// ── Folder structures ─────────────────────────────────────────────────────────

func createDynamicFolders(root string) error {
	// Dynamic projects have no pkg/ or cmd/ — just Katalog files
	dirs := []string{
		"examples/website",
		"examples/platform-namespace",
		"katalogs",
	}
	return makeDirs(root, dirs)
}

func createTypedFolders(root string) error {
	dirs := []string{
		"cmd/orkestra",
		"examples/website",
		"examples/platform-namespace",
		"katalogs",
		"pkg/runtime", // for __generated_runtime_*.go
		"pkg/hooks",   // user writes Go hooks here
	}
	return makeDirs(root, dirs)
}

func makeDirs(root string, dirs []string) error {
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			return err
		}
	}
	return nil
}

// ── File writers ──────────────────────────────────────────────────────────────

func writeMain(root string) error {
	content := `package main

import "github.com/ialexeze/orkestra/cmd/cli"

func main() {
	cli.Execute(nil, nil)
}
`
	return writeFile(root, "cmd/orkestra/main.go", content)
}

func writeGoMod(root, module string) error {
	content := fmt.Sprintf(`module %s

go 1.22

require (
	github.com/ialexeze/orkestra latest
)
`, module)
	return writeFile(root, "go.mod", content)
}

func writeEnvExample(root string) error {
	content := `# Orkestra configuration — copy to .env
KUBECONFIG=~/.kube/config
KATALOG_PATH=./examples/website/website-katalog.yaml
DEFAULT_WORKERS=2
DEFAULT_RESYNC=30s
MAX_QUEUE_DEPTH=500
HEALTH_PORT=8080
LOG_LEVEL=info
APP_NAME=orkestra
`
	return writeFile(root, ".env.example", content)
}

func writeDynamicGitignore(root string) error {
	content := `.env
`
	return writeFile(root, ".gitignore", content)
}

func writeTypedGitignore(root string) error {
	content := `.env
bin/
dist/
vendor/
# Regenerate with: ork generate runtime --katalog <path>
pkg/runtime/__generated_runtime_*.go
`
	return writeFile(root, ".gitignore", content)
}

func writeDynamicReadme(root string) error {
	name := filepath.Base(root)
	content := fmt.Sprintf(`# %s

An Orkestra operator. No Go required.

## Run

`+"```"+`bash
kubectl apply -f examples/website/website-crd.yaml
ork run --katalog examples/website/website-katalog.yaml
`+"```"+`

## Validate

`+"```"+`bash
ork validate --katalog examples/website/website-katalog.yaml
`+"```"+`

## Preview

`+"```"+`bash
ork template --katalog examples/website/website-katalog.yaml
ork template --katalog examples/website/website-katalog.yaml --graph
ork template --katalog examples/website/website-katalog.yaml --json
`+"```"+`
`, name)
	return writeFile(root, "README.md", content)
}

func writeTypedReadme(root, module string) error {
	name := filepath.Base(root)
	content := fmt.Sprintf(`# %s

An Orkestra operator with compiled Go types.

## Build and run

`+"```"+`bash
kubectl apply -f examples/website/website-crd.yaml

# Quick run without building
go run ./cmd/orkestra/ run --katalog examples/website/website-katalog.yaml

# Build binary
go build -o ./bin/%s ./cmd/orkestra/
./bin/%s run --katalog examples/website/website-katalog.yaml
`+"```"+`

## Generate runtime wiring (after changing the Katalog)

`+"```"+`bash
ork generate runtime --katalog examples/website/website-katalog.yaml
go mod tidy
`+"```"+`
`, name, name, name)
	return writeFile(root, "README.md", content)
}

func writeExampleKatalog(root string) error {
	return writeFile(root, "examples/website/website-katalog.yaml", websiteKatalogContent)
}

func writeExampleCRD(root string) error {
	return writeFile(root, "examples/website/website-crd.yaml", websiteCRDContent)
}

func writeExampleCR(root string) error {
	return writeFile(root, "examples/website/website-cr.yaml", websiteCRContent)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type initStep struct {
	name string
	fn   func() error
}

func runSteps(steps []initStep) error {
	for _, step := range steps {
		fmt.Printf("  %-50s", step.name+"...")
		if err := step.fn(); err != nil {
			fmt.Printf("%s✗%s\n", utils.ColorRed, utils.ColorReset)
			return fmt.Errorf("%s: %w", step.name, err)
		}
		fmt.Printf("%s✓%s\n", utils.ColorGreen, utils.ColorReset)
	}
	return nil
}

func writeFile(root, path, content string) error {
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0644)
}

func goModTidy(root string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}

func printBanner() {
	fmt.Printf("\n%s%s%s\n\n", utils.ColorGreen, utils.OrkestraLogoCLI, utils.ColorReset)
}

// ── Embedded starter content ──────────────────────────────────────────────────
// Embedded directly so ork init works with no internet connection.

const websiteKatalogContent = `apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-katalog
spec:
  finalizers:
    - finalizer.demo.orkestra.io/website
  crds:
    - name: website
      enabled: true
      namespaced: true
      workers: 2
      resync: 30s
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      reconciler:
        default: true
        finalizers:
          - finalizer.demo.orkestra.io/website
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              port: "{{ .spec.port }}"
              namespace: "{{ .metadata.namespace }}"
              reconcile: true
              labels:
                - key: app
                  value: "{{ .metadata.name }}"
                - key: managed-by
                  value: orkestra
          services:
            - name: "{{ .metadata.name }}-svc"
              type: "{{ .spec.serviceType }}"
              port: "80"
              targetPort: "{{ .spec.port }}"
              namespace: "{{ .metadata.namespace }}"
              reconcile: true
      queue:
        maxQueueDepth: 500
`

const websiteCRDContent = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: websites.demo.orkestra.io
spec:
  group: demo.orkestra.io
  scope: Namespaced
  names:
    plural: websites
    singular: website
    kind: Website
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required: [image]
              properties:
                image:
                  type: string
                replicas:
                  type: integer
                  default: 1
                port:
                  type: integer
                  default: 80
                serviceType:
                  type: string
                  default: ClusterIP
                  enum: [ClusterIP, NodePort, LoadBalancer]
      subresources:
        status: {}
`

const websiteCRContent = `apiVersion: demo.orkestra.io/v1alpha1
kind: Website
metadata:
  name: my-website
  namespace: default
spec:
  image: nginx:1.25
  replicas: 1
  port: 80
  serviceType: ClusterIP
`
