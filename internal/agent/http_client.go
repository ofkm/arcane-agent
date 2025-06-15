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

	regData := RegisterAgentDto{
		ID:           h.config.AgentID,
		Hostname:     hostname,
		Platform:     runtime.GOOS,
		Version:      version.GetVersion(),
		Capabilities: []string{"docker", "compose"},
		URL:          "", // Empty string if no callback URL
	}

	// Add debug logging
	jsonData, _ := json.MarshalIndent(regData, "", "  ")
	log.Printf("Registering agent with data: %s", string(jsonData))

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
		log.Printf("Task %s failed: %v", task.ID, err)
	} else {
		taskResult.Status = TaskStatusCompleted
		taskResult.Result = resultMap
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
	req.Header.Set("User-Agent", "arcane-agent/1.1.1")

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
