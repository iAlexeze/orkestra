// pkg/reconciler/run_helper_docker.go
package reconciler

import (
	"context"
	"fmt"
	"os/exec"
)

type DockerBuildResult struct {
	Error string
}

func executeDockerBuild(ctx context.Context, workingDir, dockerfile, image string) DockerBuildResult {
	cmd := exec.CommandContext(ctx,
		"docker", "build",
		"-f", dockerfile,
		"-t", image,
		workingDir,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return DockerBuildResult{
			Error: fmt.Sprintf("docker build: %v (%s)", err, out),
		}
	}

	return DockerBuildResult{}
}

type DockerPushResult struct {
	Error string
}

func executeDockerPush(ctx context.Context, image string) DockerPushResult {
	cmd := exec.CommandContext(ctx, "docker", "push", image)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return DockerPushResult{
			Error: fmt.Sprintf("docker push: %v (%s)", err, out),
		}
	}

	return DockerPushResult{}
}
