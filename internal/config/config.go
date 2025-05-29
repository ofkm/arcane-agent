package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	ArcaneHost     string
	ArcanePort     int
	AgentID        string
	ReconnectDelay time.Duration
	HeartbeatRate  time.Duration
	TLSEnabled     bool
}

func Load() (*Config, error) {
	cfg := &Config{
		ArcaneHost:     getEnv("ARCANE_HOST", "localhost"),
		ArcanePort:     getEnvInt("ARCANE_PORT", 3000),
		AgentID:        getOrCreateAgentID(),
		ReconnectDelay: getEnvDuration("RECONNECT_DELAY", 5*time.Second),
		HeartbeatRate:  getEnvDuration("HEARTBEAT_RATE", 30*time.Second),
		TLSEnabled:     getEnvBool("TLS_ENABLED", false),
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

func getOrCreateAgentID() string {
	// First check if AGENT_ID is set in environment
	if agentID := os.Getenv("AGENT_ID"); agentID != "" {
		return agentID
	}

	// Try to load from file
	agentIDFile := getAgentIDFile()
	if data, err := os.ReadFile(agentIDFile); err == nil {
		agentID := string(data)
		if agentID != "" {
			return agentID
		}
	}

	// Generate new agent ID and save it
	agentID := generateAgentID()
	saveAgentID(agentID)
	return agentID
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

func saveAgentID(agentID string) {
	agentIDFile := getAgentIDFile()

	// Create directory if it doesn't exist
	dir := filepath.Dir(agentIDFile)
	os.MkdirAll(dir, 0755)

	// Write agent ID to file
	os.WriteFile(agentIDFile, []byte(agentID), 0644)
}
