package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ofkm/arcane-agent/internal/version"
)

type Config struct {
	// Agent identity
	AgentID string `json:"agent_id"`
	Version string `json:"version"`

	// Agent API Server
	AgentListenAddress string `json:"agent_listen_address"`
	AgentPort          int    `json:"agent_port"`
	APIKey             string `json:"api_key"`
}

func Load() (*Config, error) {
	// Get or create agent ID
	agentID, err := getOrCreateAgentID()
	if err != nil {
		return nil, fmt.Errorf("failed to get agent ID: %w", err)
	}

	cfg := &Config{
		AgentID:            agentID,
		Version:            version.GetVersion(),
		AgentListenAddress: getEnv("AGENT_LISTEN_ADDRESS", "0.0.0.0"),
		AgentPort:          getEnvInt("AGENT_PORT", 3552),
		APIKey:             getEnv("API_KEY", ""),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.AgentPort <= 0 || c.AgentPort > 65535 {
		return fmt.Errorf("invalid AGENT_PORT: %d", c.AgentPort)
	}
	if c.AgentID == "" {
		return fmt.Errorf("AGENT_ID cannot be empty")
	}
	return nil
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

func getOrCreateAgentID() (string, error) {
	if agentID := os.Getenv("AGENT_ID"); agentID != "" {
		return agentID, nil
	}

	// Generate a simple agent ID based on hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return fmt.Sprintf("arcane-agent-%s", hostname), nil
}
