package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/docker"
	"github.com/ofkm/arcane-agent/internal/tasks"
	"github.com/ofkm/arcane-agent/pkg/types"
)

func TestNewHTTPClient(t *testing.T) {
	cfg := &config.Config{
		ArcaneHost: "localhost",
		ArcanePort: 3000,
		AgentID:    "test-agent",
		TLSEnabled: false,
	}

	dockerClient := docker.NewClient()
	taskManager := tasks.NewManager(dockerClient)
	httpClient := NewHTTPClient(cfg, taskManager)

	if httpClient == nil {
		t.Fatal("Expected non-nil HTTP client")
	}

	if httpClient.config != cfg {
		t.Error("Expected config to be set")
	}

	if httpClient.taskManager != taskManager {
		t.Error("Expected task manager to be set")
	}

	expectedURL := "http://localhost:3000"
	if httpClient.baseURL != expectedURL {
		t.Errorf("Expected baseURL %s, got %s", expectedURL, httpClient.baseURL)
	}
}

func TestNewHTTPClientWithTLS(t *testing.T) {
	cfg := &config.Config{
		ArcaneHost: "example.com",
		ArcanePort: 443,
		AgentID:    "test-agent",
		TLSEnabled: true,
	}

	dockerClient := docker.NewClient()
	taskManager := tasks.NewManager(dockerClient)
	httpClient := NewHTTPClient(cfg, taskManager)

	expectedURL := "https://example.com:443"
	if httpClient.baseURL != expectedURL {
		t.Errorf("Expected baseURL %s, got %s", expectedURL, httpClient.baseURL)
	}
}

func TestHTTPClientMakeRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/test":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/error":
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		ArcaneHost: "localhost",
		ArcanePort: 3000,
		AgentID:    "test-agent",
		TLSEnabled: false,
	}

	dockerClient := docker.NewClient()
	taskManager := tasks.NewManager(dockerClient)
	httpClient := NewHTTPClient(cfg, taskManager)

	// Override baseURL to use test server
	httpClient.baseURL = server.URL

	t.Run("successful request", func(t *testing.T) {
		var response map[string]string
		err := httpClient.makeRequest("GET", "/api/test", nil, &response)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if response["status"] != "ok" {
			t.Errorf("Expected status 'ok', got '%s'", response["status"])
		}
	})

	t.Run("error response", func(t *testing.T) {
		err := httpClient.makeRequest("GET", "/api/error", nil, nil)

		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})

	t.Run("request with body", func(t *testing.T) {
		body := map[string]string{"key": "value"}
		err := httpClient.makeRequest("POST", "/api/test", body, nil)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestHTTPClientStart(t *testing.T) {
	// Create test server
	var registrationCalled, heartbeatCalled, tasksCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/agents/register":
			registrationCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
		case "/api/agents/heartbeat":
			heartbeatCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/agents/test-agent/tasks":
			tasksCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]types.TaskRequest{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		ArcaneHost: "localhost",
		ArcanePort: 3000,
		AgentID:    "test-agent",
		TLSEnabled: false,
	}

	dockerClient := docker.NewClient()
	taskManager := tasks.NewManager(dockerClient)
	httpClient := NewHTTPClient(cfg, taskManager)

	// Override baseURL to use test server
	httpClient.baseURL = server.URL

	// Test registration
	err := httpClient.registerAgent()
	if err != nil {
		t.Errorf("Registration failed: %v", err)
	}
	if !registrationCalled {
		t.Error("Expected registration to be called")
	}

	// Test heartbeat
	err = httpClient.sendHeartbeat()
	if err != nil {
		t.Errorf("Heartbeat failed: %v", err)
	}
	if !heartbeatCalled {
		t.Error("Expected heartbeat to be called")
	}

	// Test task polling
	err = httpClient.pollForTasks()
	if err != nil {
		t.Errorf("Task polling failed: %v", err)
	}
	if !tasksCalled {
		t.Error("Expected tasks polling to be called")
	}
}

func TestHTTPClientStartIntegration(t *testing.T) {
	// This is a simpler integration test that just checks the client starts and stops
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/agents/register":
			json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
		case "/api/agents/heartbeat":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			json.NewEncoder(w).Encode([]types.TaskRequest{})
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		ArcaneHost: "localhost",
		ArcanePort: 3000,
		AgentID:    "test-agent",
		TLSEnabled: false,
	}

	dockerClient := docker.NewClient()
	taskManager := tasks.NewManager(dockerClient)
	httpClient := NewHTTPClient(cfg, taskManager)
	httpClient.baseURL = server.URL

	// Start with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := httpClient.Start(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestExecuteTask(t *testing.T) {
	// Create test server to receive task results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/agents/test-agent/tasks/task-123/result" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "received"})
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		ArcaneHost: "localhost",
		ArcanePort: 3000,
		AgentID:    "test-agent",
		TLSEnabled: false,
	}

	dockerClient := docker.NewClient()
	taskManager := tasks.NewManager(dockerClient)
	httpClient := NewHTTPClient(cfg, taskManager)

	// Override baseURL to use test server
	httpClient.baseURL = server.URL

	task := types.TaskRequest{
		ID:      "task-123",
		Type:    "system_info",
		Payload: map[string]interface{}{},
	}

	// Execute task (this will run in background)
	httpClient.executeTask(task)

	// Give it time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestGetHostname(t *testing.T) {
	hostname := getHostname()

	if hostname == "" {
		t.Error("Expected non-empty hostname")
	}

	if hostname == "unknown" {
		t.Log("Hostname returned 'unknown' (this might be expected in some environments)")
	}
}
