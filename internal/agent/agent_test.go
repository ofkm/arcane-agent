package agent

import (
	"testing"
	"time"

	"github.com/ofkm/arcane-agent/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		ArcaneHost:     "localhost",
		ArcanePort:     3000,
		AgentID:        "test-agent",
		ReconnectDelay: 5 * time.Second,
		HeartbeatRate:  30 * time.Second,
		TLSEnabled:     false,
	}

	agent := New(cfg)

	if agent == nil {
		t.Fatal("Expected non-nil agent")
	}

	if agent.config != cfg {
		t.Error("Expected config to be set")
	}

	if agent.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}

	if agent.dockerClient == nil {
		t.Error("Expected dockerClient to be initialized")
	}

	if agent.taskManager == nil {
		t.Error("Expected taskManager to be initialized")
	}

	if agent.shutdown == nil {
		t.Error("Expected shutdown channel to be initialized")
	}
}

func TestAgentStartStop(t *testing.T) {
	cfg := &config.Config{
		ArcaneHost:     "localhost",
		ArcanePort:     3000,
		AgentID:        "test-agent",
		ReconnectDelay: 5 * time.Second,
		HeartbeatRate:  30 * time.Second,
		TLSEnabled:     false,
	}

	agent := New(cfg)

	// Start agent in goroutine
	done := make(chan error, 1)
	go func() {
		done <- agent.Start()
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the agent
	agent.Stop()

	// Wait for start to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Expected no error from Start(), got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Agent.Start() did not complete within timeout")
	}
}

func TestAgentStop(t *testing.T) {
	cfg := &config.Config{
		ArcaneHost:     "localhost",
		ArcanePort:     3000,
		AgentID:        "test-agent",
		ReconnectDelay: 5 * time.Second,
		HeartbeatRate:  30 * time.Second,
		TLSEnabled:     false,
	}

	agent := New(cfg)

	// Calling Stop() should not panic
	agent.Stop()

	// Calling Stop() multiple times should not panic
	agent.Stop()
}
