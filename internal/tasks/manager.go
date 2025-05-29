// internal/tasks/manager.go
package tasks

import (
	"context"
	"fmt"

	"github.com/ofkm/arcane-agent/internal/docker"
)

type Manager struct {
	dockerClient *docker.Client
}

func NewManager(dockerClient *docker.Client) *Manager {
	return &Manager{
		dockerClient: dockerClient,
	}
}

func (m *Manager) ExecuteTask(taskType string, payload map[string]interface{}) (interface{}, error) {
	ctx := context.Background()

	switch taskType {
	case "docker_command":
		return m.executeDockerCommand(ctx, payload)
	case "container_start":
		return m.startContainer(ctx, payload)
	case "container_stop":
		return m.stopContainer(ctx, payload)
	case "container_restart":
		return m.restartContainer(ctx, payload)
	case "container_list":
		return m.listContainers(ctx, payload)
	case "image_pull":
		return m.pullImage(ctx, payload)
	case "image_list":
		return m.listImages(ctx, payload)
	case "system_info":
		return m.getSystemInfo(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

func (m *Manager) executeDockerCommand(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	command, ok := payload["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing command")
	}

	args := []string{}
	if argsInterface, exists := payload["args"]; exists {
		if argsList, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsList {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}
	}

	output, err := m.dockerClient.ExecuteCommand(command, args)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"output":  output,
		"command": fmt.Sprintf("docker %s %v", command, args),
	}, nil
}

func (m *Manager) startContainer(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	return m.dockerClient.StartContainer(ctx, containerID)
}

func (m *Manager) stopContainer(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	return m.dockerClient.StopContainer(ctx, containerID)
}

func (m *Manager) restartContainer(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	return m.dockerClient.RestartContainer(ctx, containerID)
}

func (m *Manager) listContainers(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	return m.dockerClient.ListContainers(ctx)
}

func (m *Manager) pullImage(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	image, ok := payload["image"].(string)
	if !ok {
		return nil, fmt.Errorf("missing image")
	}

	return m.dockerClient.PullImage(ctx, image)
}

func (m *Manager) listImages(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	return m.dockerClient.ListImages(ctx)
}

func (m *Manager) getSystemInfo(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	return m.dockerClient.GetSystemInfo(ctx)
}
