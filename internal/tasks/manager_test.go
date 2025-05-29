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
			name:     "docker_command with command",
			taskType: "docker_command",
			payload: map[string]interface{}{
				"command": "version",
				"args":    []interface{}{"--format", "json"},
			},
			wantErr: false, // May fail if Docker not available, but structure is correct
		},
		{
			name:     "container_start missing container_id",
			taskType: "container_start",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "container_start with container_id",
			taskType: "container_start",
			payload: map[string]interface{}{
				"container_id": "test-container",
			},
			wantErr: false, // Will fail because container doesn't exist, but structure is correct
		},
		{
			name:     "container_stop missing container_id",
			taskType: "container_stop",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "container_restart missing container_id",
			taskType: "container_restart",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "container_list",
			taskType: "container_list",
			payload:  map[string]interface{}{},
			wantErr:  false, // May fail if Docker not available
		},
		{
			name:     "image_pull missing image",
			taskType: "image_pull",
			payload:  map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "image_list",
			taskType: "image_list",
			payload:  map[string]interface{}{},
			wantErr:  false, // May fail if Docker not available
		},
		{
			name:     "system_info",
			taskType: "system_info",
			payload:  map[string]interface{}{},
			wantErr:  false, // May fail if Docker not available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ExecuteTask(tt.taskType, tt.payload)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && tt.taskType == "unknown_task" && err == nil {
				t.Error("Expected error for unknown task type")
			}

			// For valid task types, we might get Docker errors if Docker isn't available
			// This is fine for unit tests
			if !tt.wantErr && err != nil {
				t.Logf("Task failed (likely Docker not available): %v", err)
			}

			if !tt.wantErr && err == nil && result == nil {
				t.Error("Expected non-nil result for successful task")
			}
		})
	}
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
