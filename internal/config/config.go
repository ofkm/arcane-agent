package config

import (
	"fmt"
	"os"
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
		AgentID:        getEnv("AGENT_ID", generateAgentID()),
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

func generateAgentID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix())
}
