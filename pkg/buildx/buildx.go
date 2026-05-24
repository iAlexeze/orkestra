package buildx

import (
	"fmt"
	"io"

	"github.com/orkspace/orkestra/pkg/spinner"
)

// ComposeBuild describes how to build a container image.
type ComposeBuild struct {
	UseCompose  bool   // true → use compose build
	ComposeFile string // path to docker-compose.yaml (only when UseCompose)
	Dockerfile  string // path to Dockerfile; empty → "Dockerfile" in the build dir
}

// Builder is the interface implemented by all container builders.
type Builder interface {
	Name() string
	Build(dir, image string, compose ComposeBuild, w io.Writer) error
	Push(image string, w io.Writer) error
	Available() bool
}

//
// ────────────────────────────────────────────────────────────────
// Builder Selection + Public API
// ────────────────────────────────────────────────────────────────
//

// AllBuilders is the ordered list of supported builders.
// First available one wins.
var AllBuilders = []Builder{
	DockerBuilder{},
	PodmanBuilder{},
	BuildahBuilder{},
}

// selectBuilder returns the first available builder.
func selectBuilder() (Builder, error) {
	for _, b := range AllBuilders {
		if b.Available() {
			return b, nil
		}
	}

	// No builder found → Orkestra-style error
	return nil, fmt.Errorf(`
───────────────────────────────────────────────────────────────
❌ No container builder found

Orkestra requires one of the following tools to build images:
  • docker
  • podman
  • buildah

Install one of them and re-run your command.

Docs: https://orkestra.sh/cli/build
───────────────────────────────────────────────────────────────`)
}

// BuildImage builds an image using the first available builder.
func BuildImage(dir, image string, compose ComposeBuild, w io.Writer) error {
	b, err := selectBuilder()
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "  → Using %s builder\n", b.Name())

	spin := spinner.Start("   → Building image " + image)

	err = b.Build(dir, image, compose, w)
	if err != nil {
		spin.Failure()
		return err
	}

	spin.Success()
	return nil
}

// PushImage pushes an image using the first available builder.
func PushImage(image string, w io.Writer) error {
	b, err := selectBuilder()
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "  → Using %s builder\n", b.Name())

	spin := spinner.Start("   → Pushing image " + image)

	err = b.Push(image, w)
	if err != nil {
		spin.Failure()
		return err
	}

	spin.Success()
	return nil
}

//
// ────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────
//

func wrapErr(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}
