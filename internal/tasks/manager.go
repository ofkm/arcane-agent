// internal/tasks/manager.go
package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ofkm/arcane-agent/internal/compose"
	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/docker"
)

type Manager struct {
	dockerClient   *docker.Client
	composeManager *compose.Manager
	config         *config.Config
}

func NewManager(dockerClient *docker.Client, cfg *config.Config) *Manager {
	composeManager := compose.NewManager(cfg.ComposeBasePath)

	// Ensure base directory exists
	if err := composeManager.EnsureBaseDirectory(); err != nil {
		// Log error but don't fail initialization
		fmt.Printf("Warning: failed to create compose base directory: %v\n", err)
	}

	return &Manager{
		dockerClient:   dockerClient,
		composeManager: composeManager,
		config:         cfg,
	}
}

func (m *Manager) ExecuteTask(taskType string, payload map[string]interface{}) (interface{}, error) {
	ctx := context.Background()

	switch taskType {
	case "docker_command":
		return m.executeDockerCommand(payload)
	case "container_start":
		return m.executeContainerStart(ctx, payload)
	case "container_stop":
		return m.executeContainerStop(ctx, payload)
	case "container_restart":
		return m.executeContainerRestart(ctx, payload)
	case "container_list":
		return m.dockerClient.ListContainers(ctx)
	case "container_remove":
		return m.executeContainerRemove(ctx, payload)
	case "container_logs":
		return m.executeContainerLogs(ctx, payload)
	case "image_pull":
		return m.executeImagePull(ctx, payload)
	case "image_list":
		return m.dockerClient.ListImages(ctx)
	case "system_info":
		return m.dockerClient.GetSystemInfo(ctx)
	case "metrics":
		return m.dockerClient.GetMetrics(ctx)

	// Compose operations
	case "compose_up":
		return m.executeComposeUp(ctx, payload)
	case "compose_down":
		return m.executeComposeDown(ctx, payload)
	case "compose_ps":
		return m.executeComposePs(ctx, payload)
	case "compose_logs":
		return m.executeComposeLogs(ctx, payload)
	case "compose_deploy":
		return m.executeComposeDeploy(ctx, payload)
	case "compose_remove":
		return m.executeComposeRemove(ctx, payload)

	// Compose project management
	case "compose_create_project":
		return m.executeComposeCreateProject(payload)
	case "compose_update_project":
		return m.executeComposeUpdateProject(payload)
	case "compose_delete_project":
		return m.executeComposeDeleteProject(payload)
	case "compose_list_projects":
		return m.executeComposeListProjects()

	case "stack_list":
		return m.executeStackList(ctx)
	case "stack_services":
		return m.executeStackServices(ctx, payload)

	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

func (m *Manager) executeDockerCommand(payload map[string]interface{}) (interface{}, error) {
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

func (m *Manager) executeContainerStart(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	return m.dockerClient.StartContainer(ctx, containerID)
}

func (m *Manager) executeContainerStop(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	return m.dockerClient.StopContainer(ctx, containerID)
}

func (m *Manager) executeContainerRestart(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	return m.dockerClient.RestartContainer(ctx, containerID)
}

func (m *Manager) executeContainerRemove(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	force := false
	if f, ok := payload["force"].(bool); ok {
		force = f
	}

	return m.dockerClient.RemoveContainer(ctx, containerID, force)
}

func (m *Manager) executeContainerLogs(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing container_id")
	}

	tail := 100
	if t, ok := payload["tail"].(float64); ok {
		tail = int(t)
	}

	return m.dockerClient.GetContainerLogs(ctx, containerID, tail)
}

func (m *Manager) executeImagePull(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	var image string
	var ok bool

	if image, ok = payload["imageName"].(string); !ok {
		if image, ok = payload["image"].(string); !ok {
			return nil, fmt.Errorf("missing imageName or image")
		}
	}

	result, err := m.dockerClient.PullImage(ctx, image)
	if err != nil {
		return map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("Failed to pull image %s: %v", image, err),
		}, nil
	}

	var output string
	if resultMap, ok := result.(map[string]interface{}); ok {
		if outputStr, exists := resultMap["output"]; exists {
			output = fmt.Sprintf("%v", outputStr)
		}
	}

	return map[string]interface{}{
		"status": "completed",
		"result": map[string]interface{}{
			"output": output,
			"image":  image,
		},
	}, nil
}

// New Compose methods with project-based paths
func (m *Manager) executeComposeUp(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	projectName, composePath, err := m.getComposeProjectPath(payload)
	if err != nil {
		return nil, err
	}

	return m.dockerClient.ComposeUpWithProject(ctx, composePath, projectName)
}

func (m *Manager) executeComposeDown(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	projectName, composePath, err := m.getComposeProjectPath(payload)
	if err != nil {
		return nil, err
	}

	return m.dockerClient.ComposeDownWithProject(ctx, composePath, projectName)
}

func (m *Manager) executeComposePs(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	projectName, composePath, err := m.getComposeProjectPath(payload)
	if err != nil {
		return nil, err
	}

	return m.dockerClient.ComposePs(ctx, composePath, projectName)
}

func (m *Manager) executeComposeLogs(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	projectName, composePath, err := m.getComposeProjectPath(payload)
	if err != nil {
		return nil, err
	}

	serviceName := ""
	tail := 100

	if service, ok := payload["service_name"].(string); ok {
		serviceName = service
	}
	if t, ok := payload["tail"].(float64); ok {
		tail = int(t)
	}

	return m.dockerClient.ComposeLogs(ctx, composePath, projectName, serviceName, tail)
}

func (m *Manager) executeComposeDeploy(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	projectName, composePath, err := m.getComposeProjectPath(payload)
	if err != nil {
		return nil, err
	}

	// First bring down existing deployment
	if _, err := m.dockerClient.ComposeDownWithProject(ctx, composePath, projectName); err != nil {
		// Log but don't fail if down fails (might not exist)
	}

	// Then bring up new deployment
	return m.dockerClient.ComposeUpWithProject(ctx, composePath, projectName)
}

// New Compose project management methods
func (m *Manager) executeComposeCreateProject(payload map[string]interface{}) (interface{}, error) {
	config, err := m.parseProjectConfig(payload)
	if err != nil {
		return nil, err
	}

	if err := m.composeManager.CreateProject(config); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return map[string]interface{}{
		"status":       "created",
		"project":      config.Name,
		"path":         m.composeManager.GetProjectPath(config.Name),
		"compose_file": config.ComposeFile,
	}, nil
}

func (m *Manager) executeComposeUpdateProject(payload map[string]interface{}) (interface{}, error) {
	config, err := m.parseProjectConfig(payload)
	if err != nil {
		return nil, err
	}

	if err := m.composeManager.UpdateProject(config); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return map[string]interface{}{
		"status":       "updated",
		"project":      config.Name,
		"path":         m.composeManager.GetProjectPath(config.Name),
		"compose_file": config.ComposeFile,
	}, nil
}

func (m *Manager) executeComposeDeleteProject(payload map[string]interface{}) (interface{}, error) {
	projectName, ok := payload["project_name"].(string)
	if !ok || projectName == "" {
		return nil, fmt.Errorf("project_name is required")
	}

	if err := m.composeManager.DeleteProject(projectName); err != nil {
		return nil, fmt.Errorf("failed to delete project: %w", err)
	}

	return map[string]interface{}{
		"status":  "deleted",
		"project": projectName,
	}, nil
}

func (m *Manager) executeComposeListProjects() (interface{}, error) {
	projects, err := m.composeManager.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return map[string]interface{}{
		"projects":  projects,
		"count":     len(projects),
		"base_path": m.config.ComposeBasePath,
	}, nil
}

// executeComposeRemove removes a compose project and its files
func (m *Manager) executeComposeRemove(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	// Extract project name from payload
	projectName, ok := payload["project_name"].(string)
	if !ok || projectName == "" {
		return nil, fmt.Errorf("project_name is required")
	}

	// Check if the project exists before trying to remove it
	if !m.composeManager.ProjectExists(projectName) {
		return nil, fmt.Errorf("project %s does not exist", projectName)
	}

	// Get project path for logging
	projectPath := m.composeManager.GetProjectPath(projectName)

	// First, try to bring down the compose project if it's running
	composePath := m.composeManager.GetComposePath(projectName, "docker-compose.yml")
	if _, err := os.Stat(composePath); err == nil {
		// The compose file exists, try to bring it down
		_, _ = m.dockerClient.ComposeDown(ctx, composePath)
		// We ignore errors from ComposeDown since we want to proceed with deletion regardless
	}

	// Now delete the project files and directory
	if err := m.composeManager.DeleteProject(projectName); err != nil {
		return nil, fmt.Errorf("failed to delete project %s: %w", projectName, err)
	}

	return map[string]interface{}{
		"status":  "removed",
		"message": fmt.Sprintf("Successfully removed project %s at %s", projectName, projectPath),
		"project": map[string]interface{}{
			"id":   projectName,
			"name": projectName,
			"path": projectPath,
		},
	}, nil
}

// Helper method to parse project configuration from payload
func (m *Manager) parseProjectConfig(payload map[string]interface{}) (compose.ProjectConfig, error) {
	var config compose.ProjectConfig

	// Project name (required)
	if name, ok := payload["project_name"].(string); ok {
		config.Name = name
	} else {
		return config, fmt.Errorf("project_name is required")
	}

	// Compose content (required)
	if content, ok := payload["compose_content"].(string); ok {
		config.Content = content
	} else {
		return config, fmt.Errorf("compose_content is required")
	}

	// Optional compose file name
	if file, ok := payload["compose_file"].(string); ok {
		config.ComposeFile = file
	}

	// Optional environment variables
	if envVarsInterface, ok := payload["env_vars"]; ok {
		if envVarsMap, ok := envVarsInterface.(map[string]interface{}); ok {
			config.EnvVars = make(map[string]string)
			for key, value := range envVarsMap {
				if valueStr, ok := value.(string); ok {
					config.EnvVars[key] = valueStr
				}
			}
		}
	}

	// Optional override flag
	if override, ok := payload["override"].(bool); ok {
		config.Override = override
	}

	return config, nil
}

// Updated helper method to resolve project name and compose file path
func (m *Manager) getComposeProjectPath(payload map[string]interface{}) (string, string, error) {
	// Get project name from payload (required)
	projectName, ok := payload["project_name"].(string)
	if !ok || projectName == "" {
		return "", "", fmt.Errorf("project_name is required")
	}

	// Allow custom compose file name, default to docker-compose.yml
	composeFile := "docker-compose.yml"
	if file, ok := payload["compose_file"].(string); ok && file != "" {
		composeFile = file
	}

	// Use compose manager to get the path
	composePath := m.composeManager.GetComposePath(projectName, composeFile)

	return projectName, composePath, nil
}

func (m *Manager) executeStackList(ctx context.Context) (interface{}, error) {
	// Get all compose projects from the compose manager
	projects, err := m.composeManager.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Format as stack interface
	stacks := make([]map[string]interface{}, 0, len(projects))

	for _, project := range projects {
		projectName := project["name"].(string)

		// Create stack with basic info
		stack := map[string]interface{}{
			"id":             projectName,
			"name":           projectName,
			"path":           project["path"],
			"createdAt":      project["createdAt"],
			"updatedAt":      project["updatedAt"],
			"composeContent": project["composeContent"],
			"envContent":     project["envContent"],
			"isLegacy":       false,
			"isExternal":     false,
			"isRemote":       false,
			"agentId":        m.config.AgentID,
			"agentHostname":  getHostname(),
			"status":         "unknown", // Will update after checking services
			"serviceCount":   0,
			"runningCount":   0,
		}

		// Get services for this project to determine status
		projectName, composePath, _ := m.getComposeProjectPath(map[string]interface{}{
			"project_name": projectName,
		})

		serviceResult, err := m.dockerClient.ComposePs(ctx, composePath, projectName)
		if err == nil {
			// Parse the services output
			if resultMap, ok := serviceResult.(map[string]interface{}); ok {
				if servicesOutput, ok := resultMap["services"].(string); ok && servicesOutput != "" {
					services := m.parseComposeServicesOutput(servicesOutput)

					serviceCount := len(services)
					runningCount := 0
					for _, svc := range services {
						if state, ok := svc["state"].(map[string]interface{}); ok {
							if running, ok := state["Running"].(bool); ok && running {
								runningCount++
							}
						}
					}

					stack["serviceCount"] = serviceCount
					stack["runningCount"] = runningCount
					stack["services"] = services

					// Determine status based on service counts
					if serviceCount == 0 {
						stack["status"] = "unknown"
					} else if runningCount == 0 {
						stack["status"] = "stopped"
					} else if runningCount == serviceCount {
						stack["status"] = "running"
					} else {
						stack["status"] = "partially running"
					}
				}
			}
		}

		stacks = append(stacks, stack)
	}

	return map[string]interface{}{
		"stacks": stacks,
	}, nil
}

func (m *Manager) executeStackServices(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	projectName, ok := payload["stack_name"].(string)
	if !ok || projectName == "" {
		return nil, fmt.Errorf("stack_name is required")
	}

	projectName, composePath, err := m.getComposeProjectPath(map[string]interface{}{
		"project_name": projectName,
	})
	if err != nil {
		return nil, err
	}

	serviceResult, err := m.dockerClient.ComposePs(ctx, composePath, projectName)
	if err != nil {
		return nil, err
	}

	// Parse the services output
	services := []map[string]interface{}{}
	if resultMap, ok := serviceResult.(map[string]interface{}); ok {
		if servicesOutput, ok := resultMap["services"].(string); ok {
			services = m.parseComposeServicesOutput(servicesOutput)
		}
	}

	return map[string]interface{}{
		"stack_name": projectName,
		"services":   services,
	}, nil
}

// Helper method to parse compose ps output into service objects
func (m *Manager) parseComposeServicesOutput(output string) []map[string]interface{} {
	services := []map[string]interface{}{}

	// Split output by lines
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var serviceInfo map[string]interface{}
		if err := json.Unmarshal([]byte(line), &serviceInfo); err != nil {
			continue
		}

		// Extract service name
		serviceName := ""
		if name, ok := serviceInfo["Name"].(string); ok {
			parts := strings.Split(name, "-")
			if len(parts) > 1 {
				serviceName = parts[len(parts)-1]
			} else {
				serviceName = name
			}
		} else if service, ok := serviceInfo["Service"].(string); ok {
			serviceName = service
		} else {
			continue // Skip if no service name
		}

		// Get container ID
		containerID := ""
		if id, ok := serviceInfo["ID"].(string); ok {
			containerID = id
		} else if id, ok := serviceInfo["ContainerID"].(string); ok {
			containerID = id
		}

		// Create service entry with required format
		service := map[string]interface{}{
			"id":   containerID,
			"name": serviceName,
			"state": map[string]interface{}{
				"Running":  false,
				"Status":   "unknown",
				"ExitCode": 0,
			},
			"ports": []map[string]interface{}{},
			"networkSettings": map[string]interface{}{
				"Networks": map[string]interface{}{},
			},
		}

		// Update state
		if state, ok := serviceInfo["State"].(string); ok {
			isRunning := strings.Contains(strings.ToLower(state), "running")
			service["state"].(map[string]interface{})["Running"] = isRunning
			service["state"].(map[string]interface{})["Status"] = state
		}

		// Parse ports if available
		if ports, ok := serviceInfo["Ports"].(string); ok && ports != "" {
			portsList := []map[string]interface{}{}
			portMappings := strings.Split(ports, ", ")

			for _, portMapping := range portMappings {
				// Parse port mapping (format: "0.0.0.0:8080->80/tcp")
				parts := strings.Split(portMapping, "->")
				if len(parts) != 2 {
					continue
				}

				hostPart := strings.TrimSpace(parts[0])
				containerPart := strings.TrimSpace(parts[1])

				// Extract host port (public port)
				hostPortStr := ""
				if strings.Contains(hostPart, ":") {
					hostPortStr = strings.Split(hostPart, ":")[1]
				} else {
					hostPortStr = hostPart
				}

				// Extract container port and protocol
				containerPortAndProto := strings.Split(containerPart, "/")
				if len(containerPortAndProto) != 2 {
					continue
				}

				containerPortStr := containerPortAndProto[0]
				proto := containerPortAndProto[1]

				port := map[string]interface{}{
					"Type": proto,
				}

				if publicPort, err := strconv.Atoi(hostPortStr); err == nil {
					port["PublicPort"] = publicPort
				}

				if privatePort, err := strconv.Atoi(containerPortStr); err == nil {
					port["PrivatePort"] = privatePort
				}

				portsList = append(portsList, port)
			}

			service["ports"] = portsList
		}

		// Parse networks if available
		if networks, ok := serviceInfo["Networks"].(string); ok && networks != "" {
			networksList := strings.Split(networks, ",")
			networksMap := map[string]interface{}{}

			for _, network := range networksList {
				network = strings.TrimSpace(network)
				if network == "" {
					continue
				}

				networksMap[network] = map[string]interface{}{
					"Driver": "bridge", // Default value
				}
			}

			service["networkSettings"].(map[string]interface{})["Networks"] = networksMap
		}

		services = append(services, service)
	}

	return services
}

// Helper function to get hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
