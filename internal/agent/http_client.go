// internal/agent/http_client.go
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/tasks"
	"github.com/ofkm/arcane-agent/pkg/types"
)

type HTTPClient struct {
	config      *config.Config
	httpClient  *http.Client
	baseURL     string
	taskManager *tasks.Manager
}

func NewHTTPClient(cfg *config.Config, taskManager *tasks.Manager) *HTTPClient {
	scheme := "http"
	if cfg.TLSEnabled {
		scheme = "https"
	}

	return &HTTPClient{
		config:      cfg,
		taskManager: taskManager,
		baseURL:     fmt.Sprintf("%s://%s:%d", scheme, cfg.ArcaneHost, cfg.ArcanePort),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (h *HTTPClient) Start(ctx context.Context) error {
	// Register agent first
	if err := h.registerAgent(); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	log.Printf("Agent registered successfully")

	// Start polling loop
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("HTTP client shutting down")
			return nil
		case <-ticker.C:
			// Send heartbeat and check for tasks
			if err := h.sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}

			if err := h.pollForTasks(); err != nil {
				log.Printf("Task polling failed: %v", err)
			}
		}
	}
}

func (h *HTTPClient) registerAgent() error {
	hostname := getHostname()

	regData := map[string]interface{}{
		"agent_id":     h.config.AgentID,
		"hostname":     hostname,
		"platform":     runtime.GOOS,
		"arch":         runtime.GOARCH,
		"version":      "1.0.0",
		"capabilities": []string{"docker", "compose"},
	}

	return h.makeRequest("POST", "/api/agents/register", regData, nil)
}

func (h *HTTPClient) sendHeartbeat() error {
	heartbeatData := map[string]interface{}{
		"agent_id":  h.config.AgentID,
		"status":    "online",
		"timestamp": time.Now(),
	}

	return h.makeRequest("POST", "/api/agents/heartbeat", heartbeatData, nil)
}

func (h *HTTPClient) pollForTasks() error {
	var tasks []types.TaskRequest

	url := fmt.Sprintf("/api/agents/%s/tasks", h.config.AgentID)
	err := h.makeRequest("GET", url, nil, &tasks)

	if err != nil {
		// Check if it's a JSON parsing error (likely empty response or HTML)
		if strings.Contains(err.Error(), "invalid character") {
			log.Printf("No JSON response from tasks endpoint (likely no tasks available)")
			return nil // Don't treat this as an error
		}
		return err
	}

	// Process each task
	for _, task := range tasks {
		go h.executeTask(task)
	}

	return nil
}

func (h *HTTPClient) executeTask(task types.TaskRequest) {
	log.Printf("Executing task %s of type %s", task.ID, task.Type)

	// Execute the task using task manager
	result, err := h.taskManager.ExecuteTask(task.Type, task.Payload)

	// Send result back
	taskResult := types.TaskResult{
		TaskID: task.ID,
		Status: "completed",
		Result: result,
	}

	if err != nil {
		taskResult.Status = "failed"
		taskResult.Error = err.Error()
		log.Printf("Task %s failed: %v", task.ID, err)
	} else {
		log.Printf("Task %s completed successfully", task.ID)
	}

	url := fmt.Sprintf("/api/agents/%s/tasks/%s/result", h.config.AgentID, task.ID)
	if err := h.makeRequest("POST", url, taskResult, nil); err != nil {
		log.Printf("Failed to send task result: %v", err)
	}
}

func (h *HTTPClient) makeRequest(method, path string, body interface{}, response interface{}) error {
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, h.baseURL+path, reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "arcane-agent/1.0.0")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body first
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s - %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	if response != nil {
		// Parse the body we already read
		return json.Unmarshal(bodyBytes, response)
	}

	return nil
}

// Helper function to get hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
