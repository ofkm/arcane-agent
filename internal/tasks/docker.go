package tasks

import (
	"context"
	"fmt"
	"os/exec"
)

// DockerTaskExecutor handles Docker-specific tasks
type DockerTaskExecutor struct{}

func NewDockerTaskExecutor() *DockerTaskExecutor {
	return &DockerTaskExecutor{}
}

func (d *DockerTaskExecutor) ExecuteDockerCommand(command string, args []string) (string, error) {
	cmdArgs := append([]string{command}, args...)
	cmd := exec.Command("docker", cmdArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker command failed: %s", string(output))
	}

	return string(output), nil
}

func (d *DockerTaskExecutor) DeployStack(ctx context.Context, stackName, composeFile string) (interface{}, error) {
	cmd := exec.Command("docker", "stack", "deploy", "-c", composeFile, stackName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to deploy stack: %s", string(output))
	}

	return map[string]interface{}{
		"stack_name": stackName,
		"status":     "deployed",
		"output":     string(output),
	}, nil
}

func (d *DockerTaskExecutor) RemoveStack(ctx context.Context, stackName string) (interface{}, error) {
	cmd := exec.Command("docker", "stack", "rm", stackName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to remove stack: %s", string(output))
	}

	return map[string]interface{}{
		"stack_name": stackName,
		"status":     "removed",
		"output":     string(output),
	}, nil
}

func (d *DockerTaskExecutor) GetStackServices(ctx context.Context, stackName string) (interface{}, error) {
	cmd := exec.Command("docker", "stack", "services", stackName, "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get stack services: %s", string(output))
	}

	return map[string]interface{}{
		"stack_name": stackName,
		"services":   string(output),
	}, nil
}
