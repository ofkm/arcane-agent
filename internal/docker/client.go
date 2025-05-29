package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Client struct {
	// Simple Docker CLI client
}

func NewClient() *Client {
	return &Client{}
}

// ExecuteCommand runs any docker command with args
func (c *Client) ExecuteCommand(command string, args []string) (string, error) {
	cmdArgs := append([]string{command}, args...)
	cmd := exec.Command("docker", cmdArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker %s failed: %s", command, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// IsDockerAvailable checks if Docker is available
func (c *Client) IsDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// ListContainers gets all containers in JSON format
func (c *Client) ListContainers(ctx context.Context) (interface{}, error) {
	output, err := c.ExecuteCommand("ps", []string{"-a", "--format", "json"})
	if err != nil {
		return nil, err
	}

	// Parse JSON lines into array
	lines := strings.Split(output, "\n")
	containers := make([]interface{}, 0)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var container map[string]interface{}
		if err := json.Unmarshal([]byte(line), &container); err == nil {
			containers = append(containers, container)
		}
	}

	return map[string]interface{}{
		"containers": containers,
	}, nil
}

// StartContainer starts a container by ID or name
func (c *Client) StartContainer(ctx context.Context, containerID string) (interface{}, error) {
	output, err := c.ExecuteCommand("start", []string{containerID})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":       "started",
		"output":       output,
	}, nil
}

// StopContainer stops a container by ID or name
func (c *Client) StopContainer(ctx context.Context, containerID string) (interface{}, error) {
	output, err := c.ExecuteCommand("stop", []string{containerID})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":       "stopped",
		"output":       output,
	}, nil
}

// RestartContainer restarts a container by ID or name
func (c *Client) RestartContainer(ctx context.Context, containerID string) (interface{}, error) {
	output, err := c.ExecuteCommand("restart", []string{containerID})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":       "restarted",
		"output":       output,
	}, nil
}

// PullImage pulls a Docker image
func (c *Client) PullImage(ctx context.Context, image string) (interface{}, error) {
	output, err := c.ExecuteCommand("pull", []string{image})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image":  image,
		"status": "pulled",
		"output": output,
	}, nil
}

// ListImages gets all images in JSON format
func (c *Client) ListImages(ctx context.Context) (interface{}, error) {
	output, err := c.ExecuteCommand("images", []string{"--format", "json"})
	if err != nil {
		return nil, err
	}

	// Parse JSON lines into array
	lines := strings.Split(output, "\n")
	images := make([]interface{}, 0)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var image map[string]interface{}
		if err := json.Unmarshal([]byte(line), &image); err == nil {
			images = append(images, image)
		}
	}

	return map[string]interface{}{
		"images": images,
	}, nil
}

// GetSystemInfo gets Docker system information
func (c *Client) GetSystemInfo(ctx context.Context) (interface{}, error) {
	output, err := c.ExecuteCommand("system", []string{"info", "--format", "json"})
	if err != nil {
		return nil, err
	}

	var systemInfo map[string]interface{}
	if err := json.Unmarshal([]byte(output), &systemInfo); err != nil {
		// If JSON parsing fails, return raw output
		return map[string]interface{}{
			"system_info": output,
		}, nil
	}

	return systemInfo, nil
}

// Additional useful methods

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) (interface{}, error) {
	args := []string{"rm", containerID}
	if force {
		args = []string{"rm", "-f", containerID}
	}

	output, err := c.ExecuteCommand("rm", args[1:])
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":       "removed",
		"output":       output,
	}, nil
}

// GetContainerLogs gets logs from a container
func (c *Client) GetContainerLogs(ctx context.Context, containerID string, tail int) (interface{}, error) {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	args = append(args, containerID)

	output, err := c.ExecuteCommand("logs", args[1:])
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"logs":         output,
	}, nil
}

// ComposeUp runs docker-compose up
func (c *Client) ComposeUp(ctx context.Context, composeFile string) (interface{}, error) {
	cmd := exec.Command("docker-compose", "-f", composeFile, "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker-compose up failed: %s", string(output))
	}

	return map[string]interface{}{
		"compose_file": composeFile,
		"status":       "started",
		"output":       string(output),
	}, nil
}

// ComposeDown runs docker-compose down
func (c *Client) ComposeDown(ctx context.Context, composeFile string) (interface{}, error) {
	cmd := exec.Command("docker-compose", "-f", composeFile, "down")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker-compose down failed: %s", string(output))
	}

	return map[string]interface{}{
		"compose_file": composeFile,
		"status":       "stopped",
		"output":       string(output),
	}, nil
}
