package types

import "time"

type Message struct {
	Type      string                 `json:"type"`
	AgentID   string                 `json:"agent_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type TaskRequest struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

type TaskResult struct {
	TaskID string      `json:"task_id"`
	Status string      `json:"status"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type AgentMetrics struct {
	ContainerCount *int `json:"containerCount,omitempty"`
	ImageCount     *int `json:"imageCount,omitempty"`
	StackCount     *int `json:"stackCount,omitempty"`
	NetworkCount   *int `json:"networkCount,omitempty"`
	VolumeCount    *int `json:"volumeCount,omitempty"`
}

type HeartbeatMessage struct {
	AgentID   string        `json:"agent_id"`
	Status    string        `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
	Metrics   *AgentMetrics `json:"metrics,omitempty"`
}
