package doktor

import (
	"fmt"
	"io"
	"os/exec"
)

// ImageTag returns the full image reference for a build.
// tag defaults to the git commit short SHA when empty.
func ImageTag(registry, appName, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registry, appName, tag)
}

// Build runs docker build -t <image> <dir> and streams output to w.
func Build(dir, image string, w io.Writer) error {
	cmd := exec.Command("docker", "build", "-t", image, dir)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	return nil
}

// Push runs docker push <image> and streams output to w.
func Push(image string, w io.Writer) error {
	cmd := exec.Command("docker", "push", image)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker push: %w", err)
	}
	return nil
}
