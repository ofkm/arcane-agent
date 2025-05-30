package tasks

import (
	"testing"

	"github.com/ofkm/arcane-agent/internal/docker"
)

func TestNewManager(t *testing.T) {
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient)

	if manager == nil {
		t.Error("Expected non-nil manager")
	}

	if manager.dockerClient != dockerClient {
		t.Error("Expected docker client to be set")
	}
}

func TestExecuteTask(t *testing.T) {
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient)

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
	dockerClient := docker.NewClient()

	if !dockerClient.IsDockerAvailable() {
		t.Skip("Docker not available, skipping Docker-dependent tests")
		return
	}

	manager := NewManager(dockerClient)

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
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient)

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
			result, err := manager.executeDockerCommand(nil, tt.payload)

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
	dockerClient := docker.NewClient()
	manager := NewManager(dockerClient)

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
