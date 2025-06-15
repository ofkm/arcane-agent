package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/tasks"
	"github.com/ofkm/arcane-agent/internal/version"
	"github.com/ofkm/arcane-agent/pkg/types"
)

type WebSocketClient struct {
	config      *config.Config
	conn        *websocket.Conn
	taskManager *tasks.Manager
	mu          sync.RWMutex
	connected   bool
	reconnectCh chan struct{}
	stopCh      chan struct{}
}

type WSMessage struct {
	Type    string                 `json:"type"`
	AgentID string                 `json:"agent_id,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Update to match the actual backend message format
type WSTaskMessage struct {
	Type    string                 `json:"type"`
	TaskID  string                 `json:"task_id"`
	Command string                 `json:"command"`
	Payload map[string]interface{} `json:"payload"`
}

func NewWebSocketClient(cfg *config.Config, taskManager *tasks.Manager) *WebSocketClient {
	return &WebSocketClient{
		config:      cfg,
		taskManager: taskManager,
		reconnectCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}
}

func (ws *WebSocketClient) Start(ctx context.Context) error {
	debugLog(ws.config, "Starting WebSocket client")

	// Initial connection
	if err := ws.connect(); err != nil {
		return fmt.Errorf("failed to establish initial connection: %w", err)
	}

	// Start heartbeat and message handling
	go ws.heartbeatLoop(ctx)
	go ws.messageLoop(ctx)
	go ws.reconnectLoop(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	debugLog(ws.config, "WebSocket client shutting down")

	close(ws.stopCh)
	ws.disconnect()
	return nil
}

func (ws *WebSocketClient) connect() error {
	scheme := "ws"
	if ws.config.TLSEnabled {
		scheme = "wss"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", ws.config.ArcaneHost, ws.config.ArcanePort),
		Path:   "/ws/agents", // Your WebSocket endpoint
	}

	headers := http.Header{}
	headers.Set("X-Agent-ID", ws.config.AgentID)
	headers.Set("X-Agent-Token", ws.config.Token)
	headers.Set("User-Agent", fmt.Sprintf("arcane-agent/%s", version.GetVersion()))

	debugLog(ws.config, "Connecting to WebSocket: %s", u.String())

	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket connection failed: %w (status: %s)", err, resp.Status)
		}
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	ws.mu.Lock()
	ws.conn = conn
	ws.connected = true
	ws.mu.Unlock()

	log.Printf("WebSocket connected successfully")

	// Send initial heartbeat after connection
	go ws.sendHeartbeat()

	return nil
}

func (ws *WebSocketClient) disconnect() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.conn != nil {
		ws.conn.Close()
		ws.conn = nil
	}
	ws.connected = false
}

func (ws *WebSocketClient) isConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.connected
}

func (ws *WebSocketClient) messageLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.stopCh:
			return
		default:
			if !ws.isConnected() {
				time.Sleep(1 * time.Second)
				continue
			}

			ws.mu.RLock()
			conn := ws.conn
			ws.mu.RUnlock()

			if conn == nil {
				continue
			}

			var message json.RawMessage
			err := conn.ReadJSON(&message)
			if err != nil {
				debugLog(ws.config, "WebSocket read error: %v", err)
				ws.handleDisconnection()
				continue
			}

			ws.handleMessage(message)
		}
	}
}

func (ws *WebSocketClient) handleMessage(rawMessage json.RawMessage) {
	debugLog(ws.config, "Received WebSocket message: %s", string(rawMessage))

	// First, parse to determine message type
	var baseMessage WSMessage
	if err := json.Unmarshal(rawMessage, &baseMessage); err != nil {
		debugLog(ws.config, "Failed to parse base message: %v", err)
		return
	}

	switch baseMessage.Type {
	case "task":
		ws.handleTaskMessage(rawMessage)
	case "ping":
		ws.handlePing()
	default:
		debugLog(ws.config, "Unknown message type: %s", baseMessage.Type)
	}
}

func (ws *WebSocketClient) handleTaskMessage(rawMessage json.RawMessage) {
	var taskMessage WSTaskMessage
	if err := json.Unmarshal(rawMessage, &taskMessage); err != nil {
		debugLog(ws.config, "Failed to parse task message: %v", err)
		return
	}

	debugLog(ws.config, "Parsed task message: TaskID=%s, Command=%s, Payload=%+v",
		taskMessage.TaskID, taskMessage.Command, taskMessage.Payload)

	task := types.TaskRequest{
		ID:      taskMessage.TaskID,
		Type:    taskMessage.Command,
		Payload: taskMessage.Payload,
	}

	log.Printf("Received task via WebSocket: %s (type: %s)", task.ID, task.Type)
	go ws.executeTask(task)
}

func (ws *WebSocketClient) handlePing() {
	debugLog(ws.config, "Received ping, sending pong")
	ws.sendMessage("pong", map[string]interface{}{})
}

func (ws *WebSocketClient) executeTask(task types.TaskRequest) {
	log.Printf("Executing task %s of type %s", task.ID, task.Type)

	// Execute the task using task manager
	result, err := ws.taskManager.ExecuteTask(task.Type, task.Payload)

	// Prepare result data
	var resultMap map[string]interface{}
	if result != nil {
		if rm, ok := result.(map[string]interface{}); ok {
			resultMap = rm
		} else {
			resultMap = map[string]interface{}{"data": result}
		}
	}

	// Send result back via WebSocket
	var status AgentTaskStatus
	var errorMsg *string

	if err != nil {
		status = TaskStatusFailed
		errStr := err.Error()
		errorMsg = &errStr
		log.Printf("Task %s failed: %v", task.ID, err)
	} else {
		status = TaskStatusCompleted
		log.Printf("Task %s completed successfully", task.ID)
	}

	// Create task result message that matches what backend expects
	taskResult := map[string]interface{}{
		"task_id": task.ID,
		"status":  string(status),
		"result":  resultMap,
	}

	if errorMsg != nil {
		taskResult["error"] = *errorMsg
	}

	debugLog(ws.config, "Sending task result: %+v", taskResult)
	ws.sendMessage("task_result", taskResult)
}

func (ws *WebSocketClient) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(ws.config.HeartbeatRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.stopCh:
			return
		case <-ticker.C:
			if ws.isConnected() {
				ws.sendHeartbeat()
			}
		}
	}
}

func (ws *WebSocketClient) sendHeartbeat() {
	debugLog(ws.config, "Sending heartbeat via WebSocket")

	// Get current metrics
	metricsResult, err := ws.taskManager.ExecuteTask("metrics", map[string]interface{}{})
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
	dockerInfoResult, _ := ws.taskManager.ExecuteTask("docker_info", map[string]interface{}{})
	var dockerInfo *DockerInfo
	if dockerInfoMap, ok := dockerInfoResult.(map[string]interface{}); ok {
		dockerInfo = &DockerInfo{
			Version:    getStringFromMap(dockerInfoMap, "version"),
			Containers: getIntFromMap(dockerInfoMap, "containers"),
			Images:     getIntFromMap(dockerInfoMap, "images"),
		}
	}

	heartbeatData := map[string]interface{}{
		"status":   "online",
		"metrics":  agentMetrics,
		"docker":   dockerInfo,
		"hostname": getHostname(),
		"platform": ws.config.AgentID,
		"version":  version.GetVersion(),
	}

	ws.sendMessage("heartbeat", heartbeatData)
}

func (ws *WebSocketClient) sendMessage(msgType string, data map[string]interface{}) {
	if !ws.isConnected() {
		debugLog(ws.config, "Cannot send message: not connected")
		return
	}

	message := WSMessage{
		Type:    msgType,
		AgentID: ws.config.AgentID,
		Data:    data,
	}

	ws.mu.RLock()
	conn := ws.conn
	ws.mu.RUnlock()

	if conn == nil {
		return
	}

	if err := conn.WriteJSON(message); err != nil {
		debugLog(ws.config, "Failed to send WebSocket message: %v", err)
		ws.handleDisconnection()
	} else {
		debugLog(ws.config, "Sent WebSocket message: %s", msgType)
	}
}

func (ws *WebSocketClient) handleDisconnection() {
	debugLog(ws.config, "Handling WebSocket disconnection")

	ws.mu.Lock()
	ws.connected = false
	ws.mu.Unlock()

	// Trigger reconnection
	select {
	case ws.reconnectCh <- struct{}{}:
	default:
		// Channel already has a reconnect signal
	}
}

func (ws *WebSocketClient) reconnectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.stopCh:
			return
		case <-ws.reconnectCh:
			if !ws.isConnected() {
				log.Printf("Attempting to reconnect WebSocket...")

				ws.disconnect() // Ensure clean state

				// Wait before reconnecting
				time.Sleep(ws.config.ReconnectDelay)

				if err := ws.connect(); err != nil {
					log.Printf("Reconnection failed: %v", err)
					// Schedule another reconnect attempt
					time.AfterFunc(ws.config.ReconnectDelay, func() {
						select {
						case ws.reconnectCh <- struct{}{}:
						default:
						}
					})
				} else {
					log.Printf("WebSocket reconnected successfully")
				}
			}
		}
	}
}
