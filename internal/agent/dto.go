package agent

import "github.com/ofkm/arcane-agent/pkg/types"

// Response wrapper for API responses
type TasksResponse struct {
	Success bool                `json:"success"`
	Data    []types.TaskRequest `json:"tasks"`
	Error   string              `json:"error,omitempty"`
}

// DTO types matching the backend
type RegisterAgentDto struct {
	ID           string   `json:"id" binding:"required"`
	Hostname     string   `json:"hostname" binding:"required"`
	Platform     string   `json:"platform"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	URL          string   `json:"url"`
}

type HeartbeatDto struct {
	Status   string                 `json:"status"`
	Metrics  *AgentMetrics          `json:"metrics,omitempty"`
	Docker   *DockerInfo            `json:"docker,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SubmitTaskResultDto struct {
	Status AgentTaskStatus        `json:"status" binding:"required"`
	Result map[string]interface{} `json:"result,omitempty"`
	Error  *string                `json:"error,omitempty"`
}

// Model types matching the backend
type AgentMetrics struct {
	ContainerCount int `json:"containerCount"`
	ImageCount     int `json:"imageCount"`
	StackCount     int `json:"stackCount"`
	NetworkCount   int `json:"networkCount"`
	VolumeCount    int `json:"volumeCount"`
}

type DockerInfo struct {
	Version    string `json:"version"`
	Containers int    `json:"containers"`
	Images     int    `json:"images"`
}

type AgentTaskStatus string

const (
	TaskStatusPending   AgentTaskStatus = "pending"
	TaskStatusRunning   AgentTaskStatus = "running"
	TaskStatusCompleted AgentTaskStatus = "completed"
	TaskStatusFailed    AgentTaskStatus = "failed"
)
