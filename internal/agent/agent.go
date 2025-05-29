package agent

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ofkm/arcane-agent/internal/config"
)

type Agent struct {
	config *config.Config
	conn   *websocket.Conn
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

type Message struct {
	Type      string      `json:"type"`
	AgentID   string      `json:"agent_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

func New(cfg *config.Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	reconnectAttempts := 0
	maxReconnectAttempts := 10

	for {
		select {
		case <-ctx.Done():
			log.Printf("Agent start context cancelled")
			a.cleanup()
			return nil
		default:
			reconnectAttempts++
			log.Printf("Connection attempt %d/%d", reconnectAttempts, maxReconnectAttempts)

			if err := a.connect(); err != nil {
				log.Printf("Connection failed (attempt %d): %v", reconnectAttempts, err)

				if reconnectAttempts >= maxReconnectAttempts {
					log.Printf("Max reconnection attempts reached, giving up")
					return fmt.Errorf("failed to connect after %d attempts", maxReconnectAttempts)
				}

				time.Sleep(a.config.ReconnectDelay)
				continue
			}

			log.Printf("Connected successfully, resetting reconnect attempts")
			reconnectAttempts = 0

			if err := a.handleConnection(); err != nil {
				log.Printf("Connection error: %v", err)
			}

			log.Printf("Disconnecting...")
			a.disconnect()

			// Don't reconnect immediately, wait a bit
			log.Printf("Waiting %v before reconnecting...", a.config.ReconnectDelay)
			time.Sleep(a.config.ReconnectDelay)
		}
	}
}

func (a *Agent) connect() error {
	scheme := "ws"
	if a.config.TLSEnabled {
		scheme = "wss"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", a.config.ArcaneHost, a.config.ArcanePort),
		Path:   "/agent/connect",
	}

	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	// Send registration message
	regMsg := Message{
		Type:      "register",
		AgentID:   a.config.AgentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"hostname":     getHostname(),
			"platform":     runtime.GOOS,
			"version":      "1.0.0",
			"capabilities": []string{"docker", "compose"},
		},
	}

	return a.sendMessage(regMsg)
}

func (a *Agent) handleConnection() error {
	heartbeatTicker := time.NewTicker(a.config.HeartbeatRate)
	defer heartbeatTicker.Stop()

	// Create a context for this connection
	connCtx, connCancel := context.WithCancel(a.ctx)
	defer connCancel()

	// Channel to signal when heartbeat goroutine should stop
	heartbeatDone := make(chan struct{})

	// Start heartbeat goroutine with better error handling
	go func() {
		defer close(heartbeatDone)
		for {
			select {
			case <-connCtx.Done():
				log.Printf("Heartbeat goroutine: connection context cancelled")
				return
			case <-heartbeatTicker.C:
				log.Printf("Sending heartbeat...")
				if err := a.sendHeartbeat(); err != nil {
					log.Printf("Failed to send heartbeat: %v", err)
					connCancel()
					return
				}
				log.Printf("Heartbeat sent successfully")
			}
		}
	}()

	// Main message reading loop with better error handling
	for {
		select {
		case <-connCtx.Done():
			log.Printf("Connection context cancelled, waiting for heartbeat goroutine...")
			<-heartbeatDone // Wait for heartbeat goroutine to finish
			return nil
		default:
			a.mu.RLock()
			conn := a.conn
			a.mu.RUnlock()

			if conn == nil {
				log.Printf("Connection is nil, returning error")
				return fmt.Errorf("connection is nil")
			}

			// Set read deadline to avoid blocking forever
			deadline := time.Now().Add(5 * time.Second) // Shorter deadline
			conn.SetReadDeadline(deadline)

			var msg Message
			err := conn.ReadJSON(&msg)

			// Clear the deadline
			conn.SetReadDeadline(time.Time{})

			if err != nil {
				// Check if context was cancelled first
				if connCtx.Err() != nil {
					log.Printf("Connection cancelled during read")
					return nil
				}

				// Check for timeout
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					// Timeout is expected, continue
					continue
				}

				// Check for normal close
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("WebSocket closed normally: %v", err)
					return nil
				}

				// Check for unexpected close
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket closed unexpectedly: %v", err)
					return err
				}

				log.Printf("Error reading message: %v (type: %T)", err, err)
				return err
			}

			log.Printf("Received message: %+v", msg)
			if err := a.handleMessage(msg); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}

func (a *Agent) sendHeartbeat() error {
	msg := Message{
		Type:      "heartbeat",
		AgentID:   a.config.AgentID,
		Timestamp: time.Now(),
	}
	return a.sendMessage(msg)
}

func (a *Agent) handleMessage(msg Message) error {
	log.Printf("Received message type: %s", msg.Type)

	switch msg.Type {
	case "registered":
		log.Printf("Agent successfully registered with server")
		return nil
	case "ping":
		return a.sendPong()
	case "task":
		return a.handleTask(msg.Data)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}

	return nil
}

func (a *Agent) sendPong() error {
	msg := Message{
		Type:      "pong",
		AgentID:   a.config.AgentID,
		Timestamp: time.Now(),
	}
	return a.sendMessage(msg)
}

func (a *Agent) handleTask(data interface{}) error {
	taskData, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid task data format")
	}

	taskID, ok := taskData["id"].(string)
	if !ok {
		return fmt.Errorf("task missing ID")
	}

	taskType, ok := taskData["type"].(string)
	if !ok {
		return fmt.Errorf("task missing type")
	}

	payload, ok := taskData["payload"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("task missing payload")
	}

	log.Printf("Handling task %s of type %s", taskID, taskType)

	// Execute the task
	result, err := a.executeTask(taskType, payload)

	// Send result back
	status := "completed"
	var errorMsg string
	if err != nil {
		status = "failed"
		errorMsg = err.Error()
	}

	return a.sendTaskResult(taskID, status, result, errorMsg)
}

func (a *Agent) executeTask(taskType string, payload map[string]interface{}) (interface{}, error) {
	switch taskType {
	case "docker_command":
		return a.executeDockerCommand(payload)
	case "container_start":
		return a.startContainer(payload)
	case "container_stop":
		return a.stopContainer(payload)
	case "container_restart":
		return a.restartContainer(payload)
	case "image_pull":
		return a.pullImage(payload)
	case "stack_deploy":
		return a.deployStack(payload)
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

func (a *Agent) executeDockerCommand(payload map[string]interface{}) (interface{}, error) {
	command, ok := payload["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing command")
	}

	args := []string{}
	if argsInterface, exists := payload["args"]; exists {
		if argsList, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsList {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}
	}

	// Execute docker command
	cmdArgs := append([]string{command}, args...)
	cmd := exec.Command("docker", cmdArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker command failed: %s", string(output))
	}

	return map[string]interface{}{
		"output":  string(output),
		"command": fmt.Sprintf("docker %s", strings.Join(cmdArgs, " ")),
	}, nil
}

func (a *Agent) startContainer(payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["containerId"].(string)
	if !ok {
		return nil, fmt.Errorf("missing containerId")
	}

	cmd := exec.Command("docker", "start", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %s", string(output))
	}

	return map[string]interface{}{
		"containerId": containerID,
		"status":      "started",
		"output":      string(output),
	}, nil
}

func (a *Agent) stopContainer(payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["containerId"].(string)
	if !ok {
		return nil, fmt.Errorf("missing containerId")
	}

	cmd := exec.Command("docker", "stop", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to stop container: %s", string(output))
	}

	return map[string]interface{}{
		"containerId": containerID,
		"status":      "stopped",
		"output":      string(output),
	}, nil
}

func (a *Agent) restartContainer(payload map[string]interface{}) (interface{}, error) {
	containerID, ok := payload["containerId"].(string)
	if !ok {
		return nil, fmt.Errorf("missing containerId")
	}

	cmd := exec.Command("docker", "restart", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to restart container: %s", string(output))
	}

	return map[string]interface{}{
		"containerId": containerID,
		"status":      "restarted",
		"output":      string(output),
	}, nil
}

func (a *Agent) pullImage(payload map[string]interface{}) (interface{}, error) {
	image, ok := payload["image"].(string)
	if !ok {
		return nil, fmt.Errorf("missing image")
	}

	cmd := exec.Command("docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %s", string(output))
	}

	return map[string]interface{}{
		"image":  image,
		"status": "pulled",
		"output": string(output),
	}, nil
}

func (a *Agent) deployStack(payload map[string]interface{}) (interface{}, error) {
	stackName, ok := payload["stackName"].(string)
	if !ok {
		return nil, fmt.Errorf("missing stackName")
	}

	composeFile, ok := payload["composeFile"].(string)
	if !ok {
		return nil, fmt.Errorf("missing composeFile")
	}

	cmd := exec.Command("docker", "stack", "deploy", "-c", composeFile, stackName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to deploy stack: %s", string(output))
	}

	return map[string]interface{}{
		"stackName": stackName,
		"status":    "deployed",
		"output":    string(output),
	}, nil
}

func (a *Agent) sendTaskResult(taskID, status string, result interface{}, errorMsg string) error {
	msg := Message{
		Type:      "task_result",
		AgentID:   a.config.AgentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"task_id": taskID,
			"status":  status,
			"result":  result,
			"error":   errorMsg,
		},
	}
	return a.sendMessage(msg)
}

func (a *Agent) sendMessage(msg Message) error {
	a.mu.RLock()
	conn := a.conn
	a.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// Check if context is cancelled
	if a.ctx.Err() != nil {
		return fmt.Errorf("context cancelled")
	}

	// Set write deadline
	deadline := time.Now().Add(10 * time.Second)
	conn.SetWriteDeadline(deadline)

	err := conn.WriteJSON(msg)

	// Clear the deadline
	conn.SetWriteDeadline(time.Time{})

	if err != nil {
		log.Printf("Failed to send message type %s: %v", msg.Type, err)
		return err
	}

	log.Printf("Successfully sent message type: %s", msg.Type)
	return nil
}

func (a *Agent) disconnect() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn != nil {
		a.conn.Close()
		a.conn = nil
	}
}

func (a *Agent) cleanup() {
	a.cancel()
	a.disconnect()
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func (a *Agent) isConnectionHealthy() bool {
	a.mu.RLock()
	conn := a.conn
	a.mu.RUnlock()

	if conn == nil {
		return false
	}

	// Try to send a ping frame to check connection health
	deadline := time.Now().Add(5 * time.Second)
	err := conn.WriteControl(websocket.PingMessage, []byte{}, deadline)
	return err == nil
}
