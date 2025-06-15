package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	ArcaneHost      string        `json:"arcane_host"`
	ArcanePort      int           `json:"arcane_port"`
	AgentID         string        `json:"agent_id"`
	Token           string        `json:"token"`
	TLSEnabled      bool          `json:"tls_enabled"`
	Debug           bool          `json:"debug"`
	UseWebSocket    bool          `json:"use_websocket"`
	ReconnectDelay  time.Duration `json:"reconnect_delay"`
	HeartbeatRate   time.Duration `json:"heartbeat_rate"`
	ComposeBasePath string        `json:"compose_base_path"`
}

func Load() (*Config, error) {

	cfg := &Config{
		ArcaneHost:      getEnv("ARCANE_HOST", "localhost"),
		ArcanePort:      getEnvInt("ARCANE_PORT", 3000),
		Token:           getEnv("ARCANE_TOKEN", ""),
		TLSEnabled:      getEnvBool("TLS_ENABLED", false),
		Debug:           getEnvBool("DEBUG", false),
		UseWebSocket:    getEnvBool("USE_WEBSOCKET", true),
		ReconnectDelay:  getEnvDuration("RECONNECT_DELAY", 5*time.Second),
		HeartbeatRate:   getEnvDuration("HEARTBEAT_RATE", 30*time.Second),
		ComposeBasePath: getEnv("COMPOSE_BASE_PATH", "data/agent/compose-projects"),
	}

	// Get or generate agent ID
	agentID, err := getOrCreateAgentID()
	if err != nil {
		return nil, fmt.Errorf("failed to get agent ID: %w", err)
	}
	cfg.AgentID = agentID

	if cfg.Token == "" {
		return nil, fmt.Errorf("ARCANE_TOKEN environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getOrCreateAgentID() (string, error) {
	// First check if AGENT_ID is set in environment
	if agentID := os.Getenv("AGENT_ID"); agentID != "" {
		return agentID, nil
	}

	// Try to load from file
	agentIDFile := getAgentIDFile()
	if data, err := os.ReadFile(agentIDFile); err == nil {
		agentID := string(data)
		if agentID != "" {
			return agentID, nil
		}
	}

	// Generate new agent ID and save it
	agentID := generateAgentID()
	if err := saveAgentID(agentID); err != nil {
		return "", err
	}
	return agentID, nil
}

func generateAgentID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix())
}

func getAgentIDFile() string {
	// Store in user's home directory or current directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".agent_id"
	}
	return filepath.Join(homeDir, ".arcane-agent", "agent_id")
}

func saveAgentID(agentID string) error {
	agentIDFile := getAgentIDFile()

	// Create directory if it doesn't exist
	dir := filepath.Dir(agentIDFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write agent ID to file
	if err := os.WriteFile(agentIDFile, []byte(agentID), 0644); err != nil {
		return err
	}
	return nil
}
