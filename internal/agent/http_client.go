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
	"github.com/ofkm/arcane-agent/internal/version"
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

func (h *HTTPClient) registerAgent() error {
	hostname := getHostname()

	regData := RegisterAgentDto{
		ID:           h.config.AgentID,
		Hostname:     hostname,
		Platform:     runtime.GOOS,
		Version:      version.GetVersion(),
		Capabilities: []string{"docker", "compose"},
		URL:          "", // Empty string if no callback URL
	}

	if h.config.Debug {
		jsonData, _ := json.MarshalIndent(regData, "", "  ")
		debugLog(h.config, "Registering agent with data: %s", string(jsonData))
	}

	return h.makeRequest("POST", "/api/agents/register", regData, nil)
}

func (h *HTTPClient) sendHeartbeat() error {
	// Get current metrics
	metricsResult, err := h.taskManager.ExecuteTask("metrics", map[string]interface{}{})

	var agentMetrics *AgentMetrics
	if err == nil {
		if metricsMap, ok := metricsResult.(map[string]interface{}); ok {
			agentMetrics = &AgentMetrics{
				ContainerCount: getIntFromMap(metricsMap, "containerCount"),
				ImageCount:     getIntFromMap(metricsMap, "imageCount"),
				StackCount:     getIntFromMap(metricsMap, "stackCount"),
				NetworkCount:   getIntFromMap(metricsMap, "networkCount"),
				VolumeCount:    getIntFromMap(metricsMap, "volumeCount"),
			}
		}
	}

	// Get Docker info
	dockerInfoResult, _ := h.taskManager.ExecuteTask("docker_info", map[string]interface{}{})
	var dockerInfo *DockerInfo
	if dockerInfoMap, ok := dockerInfoResult.(map[string]interface{}); ok {
		dockerInfo = &DockerInfo{
			Version:    getStringFromMap(dockerInfoMap, "version"),
			Containers: getIntFromMap(dockerInfoMap, "containers"),
			Images:     getIntFromMap(dockerInfoMap, "images"),
		}
	}

	heartbeatData := HeartbeatDto{
		Status:  "online",
		Metrics: agentMetrics,
		Docker:  dockerInfo,
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"platform":  runtime.GOOS,
			"arch":      runtime.GOARCH,
		},
	}

	debugLog(h.config, "Sending heartbeat")
	url := fmt.Sprintf("/api/agents/%s/heartbeat", h.config.AgentID)
	return h.makeRequest("POST", url, heartbeatData, nil)
}

func (h *HTTPClient) pollForTasks() error {
	debugLog(h.config, "Polling for tasks for agent %s", h.config.AgentID)

	var response TasksResponse

	url := fmt.Sprintf("/api/agents/%s/tasks/pending", h.config.AgentID)
	err := h.makeRequest("GET", url, nil, &response)

	if err != nil {
		debugLog(h.config, "Error making request to tasks endpoint: %v", err)
		// Check if it's a JSON parsing error (likely empty response or HTML)
		if strings.Contains(err.Error(), "invalid character") {
			debugLog(h.config, "No JSON response from tasks endpoint (likely no tasks available)")
			return nil // Don't treat this as an error
		}
		return err
	}

	debugLog(h.config, "Tasks response received - Success: %t, Error: %s, Data length: %d",
		response.Success, response.Error, len(response.Data))

	// Check if the response indicates success
	if !response.Success {
		if response.Error != "" {
			debugLog(h.config, "Backend error getting tasks: %s", response.Error)
		}
		return nil // Don't treat backend errors as fatal
	}

	// Process each task
	if len(response.Data) > 0 {
		log.Printf("Retrieved %d pending tasks", len(response.Data)) // Keep this as regular log
		for i, task := range response.Data {
			debugLog(h.config, "Task %d: ID=%s, Type=%s, Payload=%+v", i, task.ID, task.Type, task.Payload)
			go h.executeTask(task)
		}
	} else {
		debugLog(h.config, "No pending tasks found")
	}

	return nil
}

func (h *HTTPClient) executeTask(task types.TaskRequest) {
	log.Printf("Executing task %s of type %s", task.ID, task.Type) // Keep this as regular log

	// Execute the task using task manager
	result, err := h.taskManager.ExecuteTask(task.Type, task.Payload)

	// Prepare result data
	var resultMap map[string]interface{}
	if result != nil {
		if rm, ok := result.(map[string]interface{}); ok {
			resultMap = rm
		} else {
			resultMap = map[string]interface{}{"data": result}
		}
	}

	// Send result back using SubmitTaskResultDto
	var taskResult SubmitTaskResultDto
	var errorMsg *string

	if err != nil {
		taskResult.Status = TaskStatusFailed
		errStr := err.Error()
		errorMsg = &errStr
		taskResult.Error = errorMsg
		log.Printf("Task %s failed: %v", task.ID, err) // Keep this as regular log
	} else {
		taskResult.Status = TaskStatusCompleted
		taskResult.Result = resultMap
		log.Printf("Task %s completed successfully", task.ID) // Keep this as regular log
	}

	url := fmt.Sprintf("/api/agents/%s/tasks/%s/result", h.config.AgentID, task.ID)
	if err := h.makeRequest("POST", url, taskResult, nil); err != nil {
		log.Printf("Failed to send task result: %v", err) // Keep this as regular log
	}
}

func (h *HTTPClient) makeRequest(method, path string, body interface{}, response interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
		debugLog(h.config, "Making %s request to %s with body: %s", method, path, string(jsonData))
	} else {
		debugLog(h.config, "Making %s request to %s", method, path)
	}

	url := h.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("arcane-agent/%s", version.GetVersion()))

	// Add authentication token as X-Agent-Token header
	if h.config.Token != "" {
		req.Header.Set("X-Agent-Token", h.config.Token)
	}

	debugLog(h.config, "Request headers: %+v", req.Header)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		debugLog(h.config, "Request failed: %v", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	debugLog(h.config, "Response status: %s (%d)", resp.Status, resp.StatusCode)
	debugLog(h.config, "Response headers: %+v", resp.Header)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLog(h.config, "Failed to read response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	debugLog(h.config, "Response body: %s", string(respBody))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s - %s", resp.StatusCode, resp.Status, string(respBody))
	}

	if response != nil && len(respBody) > 0 {
		debugLog(h.config, "Attempting to unmarshal response into type: %T", response)
		if err := json.Unmarshal(respBody, response); err != nil {
			debugLog(h.config, "Failed to unmarshal response: %v", err)
			debugLog(h.config, "Response body was: %s", string(respBody))
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
		debugLog(h.config, "Successfully unmarshaled response: %+v", response)
	}

	return nil
}

func (h *HTTPClient) RegisterAgent() error {
	return h.registerAgent()
}

// Separate polling logic from Start method
func (h *HTTPClient) startPolling(ctx context.Context) error {
	log.Printf("Starting HTTP polling mode")

	// Start polling loops
	heartbeatTicker := time.NewTicker(h.config.HeartbeatRate)
	taskTicker := time.NewTicker(5 * time.Second) // Poll every 5 seconds

	defer heartbeatTicker.Stop()
	defer taskTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			debugLog(h.config, "HTTP polling shutting down")
			return nil
		case <-heartbeatTicker.C:
			debugLog(h.config, "Heartbeat timer triggered")
			if err := h.sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			} else {
				debugLog(h.config, "Heartbeat sent successfully")
			}
		case <-taskTicker.C:
			debugLog(h.config, "Task polling timer triggered")
			if err := h.pollForTasks(); err != nil {
				log.Printf("Task polling failed: %v", err)
			} else {
				debugLog(h.config, "Task polling completed")
			}
		}
	}
}

func (h *HTTPClient) Start(ctx context.Context) error {
	// Register agent first
	if err := h.registerAgent(); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	log.Printf("Agent registered successfully")

	// Start polling
	return h.startPolling(ctx)
}

// Helper function to get hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
