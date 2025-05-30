package tasks

import (
	"testing"

	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/docker"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient, cfg)

	if manager == nil {
		t.Error("Expected non-nil manager")
	}

	if manager.dockerClient != dockerClient {
		t.Error("Expected docker client to be set")
	}

	if manager.config != cfg {
		t.Error("Expected config to be set")
	}
}

func TestExecuteTask(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient, cfg)

	// Test structure validation (doesn't require Docker)
	tests := []struct {
		name     string
		taskType string
		payload  map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "unknown task type",
			taskType: "unknown_task",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "docker_command missing command",
			taskType: "docker_command",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "container_start missing container_id",
			taskType: "container_start",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "container_stop missing container_id",
			taskType: "container_stop",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "compose_up missing project_name",
			taskType: "compose_up",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "compose_down missing project_name",
			taskType: "compose_down",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "compose_ps missing project_name",
			taskType: "compose_ps",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ExecuteTask(tt.taskType, tt.payload)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Logf("Task failed (might be expected): %v", err)
			}

			// For unknown task type, we should definitely get an error
			if tt.taskType == "unknown_task" && err == nil {
				t.Error("Expected error for unknown task type")
			}

			if err == nil && result == nil {
				t.Error("Expected non-nil result for successful task")
			}
		})
	}
}

// Test Docker operations only if Docker is available
func TestExecuteTaskWithDocker(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()

	if !dockerClient.IsDockerAvailable() {
		t.Skip("Docker not available, skipping Docker-dependent tests")
		return
	}

	manager := NewManager(dockerClient, cfg)

	t.Run("docker version command", func(t *testing.T) {
		result, err := manager.ExecuteTask("docker_command", map[string]interface{}{
			"command": "version",
			"args":    []interface{}{"--format", "json"},
		})

		if err != nil {
			t.Logf("Docker command failed: %v", err)
			return
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}
	})

	t.Run("list containers", func(t *testing.T) {
		result, err := manager.ExecuteTask("container_list", map[string]interface{}{})

		if err != nil {
			t.Logf("Container list failed: %v", err)
			return
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

func TestExecuteDockerCommand(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient, cfg)

	tests := []struct {
		name    string
		payload map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing command",
			payload: map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "command without args",
			payload: map[string]interface{}{
				"command": "version",
			},
			wantErr: false, // May fail if Docker not available
		},
		{
			name: "command with args",
			payload: map[string]interface{}{
				"command": "version",
				"args":    []interface{}{"--format", "json"},
			},
			wantErr: false, // May fail if Docker not available
		},
		{
			name: "command with invalid args type",
			payload: map[string]interface{}{
				"command": "version",
				"args":    "not_an_array",
			},
			wantErr: false, // Args will be ignored, command will run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.executeDockerCommand(tt.payload)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Logf("Docker command failed (likely Docker not available): %v", err)
			}

			if !tt.wantErr && err == nil {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Error("Expected result to be a map")
					return
				}

				if _, exists := resultMap["output"]; !exists {
					t.Error("Expected 'output' key in result")
				}

				if _, exists := resultMap["command"]; !exists {
					t.Error("Expected 'command' key in result")
				}
			}
		})
	}
}

func TestExecuteMetricsTask(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient, cfg)

	result, err := manager.ExecuteTask("metrics", map[string]interface{}{})

	// May fail if Docker not available, but structure should be correct
	if err != nil {
		t.Logf("Metrics task failed (likely Docker not available): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected non-nil result for metrics task")
		return
	}

	metricsMap, ok := result.(map[string]interface{})
	if !ok {
		t.Error("Expected metrics result to be a map")
		return
	}

	expectedKeys := []string{"containerCount", "imageCount", "stackCount", "networkCount", "volumeCount"}
	for _, key := range expectedKeys {
		if _, exists := metricsMap[key]; !exists {
			t.Errorf("Expected '%s' key in metrics", key)
		}
	}
}

func TestGetComposeProjectPath(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient, cfg)

	tests := []struct {
		name            string
		payload         map[string]interface{}
		expectedProject string
		expectedPath    string
		expectError     bool
	}{
		{
			name: "basic project",
			payload: map[string]interface{}{
				"project_name": "web-app",
			},
			expectedProject: "web-app",
			expectedPath:    "/opt/compose-projects/web-app/docker-compose.yml",
			expectError:     false,
		},
		{
			name: "project with custom compose file",
			payload: map[string]interface{}{
				"project_name": "api-gateway",
				"compose_file": "docker-compose.prod.yml",
			},
			expectedProject: "api-gateway",
			expectedPath:    "/opt/compose-projects/api-gateway/docker-compose.prod.yml",
			expectError:     false,
		},
		{
			name:        "missing project name",
			payload:     map[string]interface{}{},
			expectError: true,
		},
		{
			name: "empty project name",
			payload: map[string]interface{}{
				"project_name": "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectName, composePath, err := manager.getComposeProjectPath(tt.payload)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if projectName != tt.expectedProject {
				t.Errorf("Expected project name %s, got %s", tt.expectedProject, projectName)
			}

			if composePath != tt.expectedPath {
				t.Errorf("Expected compose path %s, got %s", tt.expectedPath, composePath)
			}
		})
	}
}

func TestExecuteComposeTaskStructure(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient, cfg)

	tests := []struct {
		name     string
		taskType string
		payload  map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "compose_up missing project name",
			taskType: "compose_up",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "compose_up with project name",
			taskType: "compose_up",
			payload: map[string]interface{}{
				"project_name": "test-project",
			},
			wantErr: true, // Will fail because compose file doesn't exist
		},
		{
			name:     "compose_ps missing project name",
			taskType: "compose_ps",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "compose_logs with service",
			taskType: "compose_logs",
			payload: map[string]interface{}{
				"project_name": "test-project",
				"service_name": "web",
				"tail":         50,
			},
			wantErr: true, // Will fail because compose file doesn't exist
		},
		{
			name:     "compose_deploy with project",
			taskType: "compose_deploy",
			payload: map[string]interface{}{
				"project_name": "test-project",
			},
			wantErr: true, // Will fail because compose file doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ExecuteTask(tt.taskType, tt.payload)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Log result for debugging
			t.Logf("Task %s result: %v, error: %v", tt.taskType, result, err)
		})
	}
}

// Test compose operations with Docker (if available)
func TestExecuteComposeTaskWithDocker(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/tmp/test-compose",
	}
	dockerClient := docker.NewClient()

	if !dockerClient.IsDockerAvailable() {
		t.Skip("Docker not available, skipping Docker-dependent tests")
		return
	}

	manager := NewManager(dockerClient, cfg)

	t.Run("compose operations require project name", func(t *testing.T) {
		composeTasks := []string{"compose_up", "compose_down", "compose_ps", "compose_logs", "compose_deploy"}

		for _, taskType := range composeTasks {
			t.Run(taskType, func(t *testing.T) {
				// Test without project name (should fail)
				_, err := manager.ExecuteTask(taskType, map[string]interface{}{})
				if err == nil {
					t.Errorf("Expected error for %s without project_name", taskType)
				}

				// Test with project name (will fail because compose file doesn't exist, but error should be different)
				_, err = manager.ExecuteTask(taskType, map[string]interface{}{
					"project_name": "nonexistent-project",
				})
				if err == nil {
					t.Logf("Unexpectedly succeeded for %s with nonexistent project", taskType)
				} else {
					t.Logf("Expected failure for %s with nonexistent project: %v", taskType, err)
				}
			})
		}
	})
}

// Test the updated ExecuteTask signature compatibility
func TestExecuteTaskSignature(t *testing.T) {
	cfg := &config.Config{
		ComposeBasePath: "/opt/compose-projects",
	}
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient, cfg)

	// Test that ExecuteTask accepts the expected parameters
	result, err := manager.ExecuteTask("unknown_task", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for unknown task type")
	}

	if result != nil {
		t.Error("Expected nil result for failed task")
	}

	// Verify the error message
	expectedErrorMsg := "unknown task type: unknown_task"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}
