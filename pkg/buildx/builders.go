package buildx

import (
	"fmt"
	"io"
	"os/exec"
)

//
// ────────────────────────────────────────────────────────────────
// Docker Builder
// ────────────────────────────────────────────────────────────────
//

type DockerBuilder struct{}

func (DockerBuilder) Name() string { return "docker" }

func (DockerBuilder) Available() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func (DockerBuilder) Build(dir, image string, compose ComposeBuild, w io.Writer) error {
	if compose.UseCompose {
		cmd := exec.Command("docker", "compose", "-f", compose.ComposeFile, "build")
		cmd.Stdout = w
		cmd.Stderr = w
		return wrapErr("docker compose build", cmd.Run())
	}

	cmd := exec.Command("docker", "build", "-t", image, dir)
	cmd.Stdout = w
	cmd.Stderr = w
	return wrapErr("docker build", cmd.Run())
}

func (DockerBuilder) Push(image string, w io.Writer) error {
	cmd := exec.Command("docker", "push", image)
	cmd.Stdout = w
	cmd.Stderr = w
	return wrapErr("docker push", cmd.Run())
}

//
// ────────────────────────────────────────────────────────────────
// Podman Builder
// ────────────────────────────────────────────────────────────────
//

type PodmanBuilder struct{}

func (PodmanBuilder) Name() string { return "podman" }

func (PodmanBuilder) Available() bool {
	_, err := exec.LookPath("podman")
	return err == nil
}

func (PodmanBuilder) Build(dir, image string, compose ComposeBuild, w io.Writer) error {
	if compose.UseCompose {
		// podman-compose is a separate binary
		composeCmd := "podman-compose"
		if _, err := exec.LookPath(composeCmd); err != nil {
			return fmt.Errorf("podman-compose not found: install podman-compose or disable compose mode")
		}
		cmd := exec.Command(composeCmd, "-f", compose.ComposeFile, "build")
		cmd.Stdout = w
		cmd.Stderr = w
		return wrapErr("podman-compose build", cmd.Run())
	}

	cmd := exec.Command("podman", "build", "-t", image, dir)
	cmd.Stdout = w
	cmd.Stderr = w
	return wrapErr("podman build", cmd.Run())
}

func (PodmanBuilder) Push(image string, w io.Writer) error {
	cmd := exec.Command("podman", "push", image)
	cmd.Stdout = w
	cmd.Stderr = w
	return wrapErr("podman push", cmd.Run())
}

//
// ────────────────────────────────────────────────────────────────
// Buildah Builder
// ────────────────────────────────────────────────────────────────
//

type BuildahBuilder struct{}

func (BuildahBuilder) Name() string { return "buildah" }

func (BuildahBuilder) Available() bool {
	_, err := exec.LookPath("buildah")
	return err == nil
}

func (BuildahBuilder) Build(dir, image string, compose ComposeBuild, w io.Writer) error {
	if compose.UseCompose {
		return fmt.Errorf("buildah does not support compose builds")
	}

	cmd := exec.Command("buildah", "build", "-t", image, dir)
	cmd.Stdout = w
	cmd.Stderr = w
	return wrapErr("buildah build", cmd.Run())
}

func (BuildahBuilder) Push(image string, w io.Writer) error {
	cmd := exec.Command("buildah", "push", image)
	cmd.Stdout = w
	cmd.Stderr = w
	return wrapErr("buildah push", cmd.Run())
}
