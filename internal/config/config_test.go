package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalEnv := map[string]string{
		"ARCANE_HOST":     os.Getenv("ARCANE_HOST"),
		"ARCANE_PORT":     os.Getenv("ARCANE_PORT"),
		"AGENT_ID":        os.Getenv("AGENT_ID"),
		"RECONNECT_DELAY": os.Getenv("RECONNECT_DELAY"),
		"HEARTBEAT_RATE":  os.Getenv("HEARTBEAT_RATE"),
		"TLS_ENABLED":     os.Getenv("TLS_ENABLED"),
	}

	// Clean env vars
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear all env vars
	for key := range originalEnv {
		os.Unsetenv(key)
	}

	t.Run("default values", func(t *testing.T) {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}

		if cfg.ArcaneHost != "localhost" {
			t.Errorf("Expected ArcaneHost 'localhost', got '%s'", cfg.ArcaneHost)
		}

		if cfg.ArcanePort != 3000 {
			t.Errorf("Expected ArcanePort 3000, got %d", cfg.ArcanePort)
		}

		if cfg.ReconnectDelay != 5*time.Second {
			t.Errorf("Expected ReconnectDelay 5s, got %v", cfg.ReconnectDelay)
		}

		if cfg.HeartbeatRate != 30*time.Second {
			t.Errorf("Expected HeartbeatRate 30s, got %v", cfg.HeartbeatRate)
		}

		if cfg.TLSEnabled != false {
			t.Errorf("Expected TLSEnabled false, got %v", cfg.TLSEnabled)
		}

		if cfg.AgentID == "" {
			t.Error("Expected AgentID to be generated, got empty string")
		}
	})

	t.Run("custom values from env", func(t *testing.T) {
		os.Setenv("ARCANE_HOST", "example.com")
		os.Setenv("ARCANE_PORT", "8080")
		os.Setenv("AGENT_ID", "test-agent-123")
		os.Setenv("RECONNECT_DELAY", "10s")
		os.Setenv("HEARTBEAT_RATE", "60s")
		os.Setenv("TLS_ENABLED", "true")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}

		if cfg.ArcaneHost != "example.com" {
			t.Errorf("Expected ArcaneHost 'example.com', got '%s'", cfg.ArcaneHost)
		}

		if cfg.ArcanePort != 8080 {
			t.Errorf("Expected ArcanePort 8080, got %d", cfg.ArcanePort)
		}

		if cfg.AgentID != "test-agent-123" {
			t.Errorf("Expected AgentID 'test-agent-123', got '%s'", cfg.AgentID)
		}

		if cfg.ReconnectDelay != 10*time.Second {
			t.Errorf("Expected ReconnectDelay 10s, got %v", cfg.ReconnectDelay)
		}

		if cfg.HeartbeatRate != 60*time.Second {
			t.Errorf("Expected HeartbeatRate 60s, got %v", cfg.HeartbeatRate)
		}

		if cfg.TLSEnabled != true {
			t.Errorf("Expected TLSEnabled true, got %v", cfg.TLSEnabled)
		}
	})
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default when env not set",
			key:          "NONEXISTENT_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "returns env value when valid int",
			key:          "TEST_INT",
			defaultValue: 42,
			envValue:     "123",
			expected:     123,
		},
		{
			name:         "returns default when env not set",
			key:          "NONEXISTENT_INT",
			defaultValue: 42,
			envValue:     "",
			expected:     42,
		},
		{
			name:         "returns default when env invalid",
			key:          "INVALID_INT",
			defaultValue: 42,
			envValue:     "not_a_number",
			expected:     42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "returns env value when valid duration",
			key:          "TEST_DURATION",
			defaultValue: 5 * time.Second,
			envValue:     "10s",
			expected:     10 * time.Second,
		},
		{
			name:         "returns default when env not set",
			key:          "NONEXISTENT_DURATION",
			defaultValue: 5 * time.Second,
			envValue:     "",
			expected:     5 * time.Second,
		},
		{
			name:         "returns default when env invalid",
			key:          "INVALID_DURATION",
			defaultValue: 5 * time.Second,
			envValue:     "not_a_duration",
			expected:     5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvDuration(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateAgentID(t *testing.T) {
	agentID := generateAgentID()

	if agentID == "" {
		t.Error("Expected non-empty agent ID")
	}

	if len(agentID) < 10 {
		t.Errorf("Expected agent ID to be at least 10 characters, got %d", len(agentID))
	}

	// Should start with "agent-"
	if agentID[:6] != "agent-" {
		t.Errorf("Expected agent ID to start with 'agent-', got %s", agentID)
	}
}

func TestGetOrCreateAgentID(t *testing.T) {
	// Save original env
	originalAgentID := os.Getenv("AGENT_ID")
	defer func() {
		if originalAgentID == "" {
			os.Unsetenv("AGENT_ID")
		} else {
			os.Setenv("AGENT_ID", originalAgentID)
		}
	}()

	t.Run("returns env AGENT_ID when set", func(t *testing.T) {
		os.Setenv("AGENT_ID", "test-env-agent")
		agentID := getOrCreateAgentID()
		if agentID != "test-env-agent" {
			t.Errorf("Expected 'test-env-agent', got '%s'", agentID)
		}
	})

	t.Run("generates new agent ID when env not set", func(t *testing.T) {
		os.Unsetenv("AGENT_ID")

		// Clean up any existing agent ID file
		agentIDFile := getAgentIDFile()
		os.Remove(agentIDFile)
		os.RemoveAll(filepath.Dir(agentIDFile))

		agentID := getOrCreateAgentID()
		if agentID == "" {
			t.Error("Expected non-empty agent ID")
		}

		if agentID[:6] != "agent-" {
			t.Errorf("Expected agent ID to start with 'agent-', got %s", agentID)
		}
	})
}
