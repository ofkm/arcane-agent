package docker

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("Expected non-nil client")
	}
}

func TestExecuteCommand(t *testing.T) {
	client := NewClient()

	t.Run("simple command", func(t *testing.T) {
		// Test with docker version command
		output, err := client.ExecuteCommand("version", []string{"--format", "json"})

		// This will fail if Docker isn't installed, which is expected in CI
		if err != nil {
			t.Logf("Docker not available (expected in CI): %v", err)
			return
		}

		if output == "" {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("invalid command", func(t *testing.T) {
		_, err := client.ExecuteCommand("invalid-command", []string{})
		if err == nil {
			t.Error("Expected error for invalid command")
		}
	})
}

func TestIsDockerAvailable(t *testing.T) {
	client := NewClient()

	// This test will pass/fail based on whether Docker is installed
	available := client.IsDockerAvailable()
	t.Logf("Docker available: %v", available)

	// We don't assert true/false since Docker may not be available in CI
	// This is more of an integration test
}

func TestListContainers(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	result, err := client.ListContainers(ctx)

	// Skip test if Docker not available
	if err != nil {
		t.Logf("Docker not available: %v", err)
		return
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Error("Expected result to be a map")
		return
	}

	if _, exists := resultMap["containers"]; !exists {
		t.Error("Expected 'containers' key in result")
	}
}

func TestStartContainer(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test with a non-existent container (should fail)
	_, err := client.StartContainer(ctx, "non-existent-container")
	if err == nil {
		t.Error("Expected error for non-existent container")
	}
}

func TestStopContainer(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test with a non-existent container (should fail)
	_, err := client.StopContainer(ctx, "non-existent-container")
	if err == nil {
		t.Error("Expected error for non-existent container")
	}
}

func TestRestartContainer(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test with a non-existent container (should fail)
	_, err := client.RestartContainer(ctx, "non-existent-container")
	if err == nil {
		t.Error("Expected error for non-existent container")
	}
}

func TestGetSystemInfo(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	result, err := client.GetSystemInfo(ctx)

	// Skip test if Docker not available
	if err != nil {
		t.Logf("Docker not available: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestRemoveContainer(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test with a non-existent container (should fail)
	_, err := client.RemoveContainer(ctx, "non-existent-container", false)
	if err == nil {
		t.Error("Expected error for non-existent container")
	}

	// Test with force flag
	_, err = client.RemoveContainer(ctx, "non-existent-container", true)
	if err == nil {
		t.Error("Expected error for non-existent container even with force")
	}
}

func TestGetContainerLogs(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test with a non-existent container (should fail)
	_, err := client.GetContainerLogs(ctx, "non-existent-container", 10)
	if err == nil {
		t.Error("Expected error for non-existent container")
	}
}
