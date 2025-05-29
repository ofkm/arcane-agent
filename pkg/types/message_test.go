package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageSerialization(t *testing.T) {
	now := time.Now()
	msg := Message{
		Type:      "test",
		AgentID:   "agent-123",
		Timestamp: now,
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if unmarshaled.Type != msg.Type {
		t.Errorf("Expected Type %s, got %s", msg.Type, unmarshaled.Type)
	}

	if unmarshaled.AgentID != msg.AgentID {
		t.Errorf("Expected AgentID %s, got %s", msg.AgentID, unmarshaled.AgentID)
	}

	if unmarshaled.Timestamp.Unix() != msg.Timestamp.Unix() {
		t.Errorf("Expected Timestamp %v, got %v", msg.Timestamp, unmarshaled.Timestamp)
	}

	if len(unmarshaled.Data) != len(msg.Data) {
		t.Errorf("Expected Data length %d, got %d", len(msg.Data), len(unmarshaled.Data))
	}
}

func TestTaskRequestSerialization(t *testing.T) {
	task := TaskRequest{
		ID:   "task-123",
		Type: "docker_command",
		Payload: map[string]interface{}{
			"command": "version",
			"args":    []string{"--format", "json"},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task request: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled TaskRequest
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal task request: %v", err)
	}

	if unmarshaled.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, unmarshaled.ID)
	}

	if unmarshaled.Type != task.Type {
		t.Errorf("Expected Type %s, got %s", task.Type, unmarshaled.Type)
	}

	if len(unmarshaled.Payload) != len(task.Payload) {
		t.Errorf("Expected Payload length %d, got %d", len(task.Payload), len(unmarshaled.Payload))
	}
}

func TestTaskResultSerialization(t *testing.T) {
	result := TaskResult{
		TaskID: "task-123",
		Status: "completed",
		Result: map[string]interface{}{
			"output": "docker version output",
		},
		Error: "",
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal task result: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled TaskResult
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal task result: %v", err)
	}

	if unmarshaled.TaskID != result.TaskID {
		t.Errorf("Expected TaskID %s, got %s", result.TaskID, unmarshaled.TaskID)
	}

	if unmarshaled.Status != result.Status {
		t.Errorf("Expected Status %s, got %s", result.Status, unmarshaled.Status)
	}

	if unmarshaled.Error != result.Error {
		t.Errorf("Expected Error %s, got %s", result.Error, unmarshaled.Error)
	}
}

func TestTaskResultWithError(t *testing.T) {
	result := TaskResult{
		TaskID: "task-123",
		Status: "failed",
		Result: nil,
		Error:  "container not found",
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal task result: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled TaskResult
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal task result: %v", err)
	}

	if unmarshaled.Status != "failed" {
		t.Errorf("Expected Status 'failed', got %s", unmarshaled.Status)
	}

	if unmarshaled.Error != "container not found" {
		t.Errorf("Expected Error 'container not found', got %s", unmarshaled.Error)
	}

	if unmarshaled.Result != nil {
		t.Errorf("Expected Result to be nil, got %v", unmarshaled.Result)
	}
}
