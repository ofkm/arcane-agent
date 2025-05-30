package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Manager struct {
	basePath string
}

type ProjectConfig struct {
	Name        string            `json:"name"`
	ComposeFile string            `json:"compose_file,omitempty"` // Optional, defaults to docker-compose.yml
	Content     string            `json:"content"`                // Docker compose YAML content
	EnvVars     map[string]string `json:"env_vars,omitempty"`     // Environment variables for .env file
	Override    bool              `json:"override,omitempty"`     // Whether to override existing files
}

func NewManager(basePath string) *Manager {
	return &Manager{
		basePath: basePath,
	}
}

// EnsureBaseDirectory creates the base compose directory if it doesn't exist
func (m *Manager) EnsureBaseDirectory() error {
	if err := os.MkdirAll(m.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", m.basePath, err)
	}
	return nil
}

// CreateProject creates a new compose project directory with files
func (m *Manager) CreateProject(config ProjectConfig) error {
	if config.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if config.Content == "" {
		return fmt.Errorf("compose content is required")
	}

	// Set default compose file name
	if config.ComposeFile == "" {
		config.ComposeFile = "docker-compose.yml"
	}

	projectPath := filepath.Join(m.basePath, config.Name)

	// Create project directory
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory %s: %w", projectPath, err)
	}

	// Create compose file
	composeFilePath := filepath.Join(projectPath, config.ComposeFile)
	if err := m.writeFileIfNotExists(composeFilePath, config.Content, config.Override); err != nil {
		return fmt.Errorf("failed to create compose file: %w", err)
	}

	// Create .env file if env vars provided
	if len(config.EnvVars) > 0 {
		envFilePath := filepath.Join(projectPath, ".env")
		envContent := m.generateEnvContent(config.EnvVars)
		if err := m.writeFileIfNotExists(envFilePath, envContent, config.Override); err != nil {
			return fmt.Errorf("failed to create .env file: %w", err)
		}
	}

	return nil
}

// UpdateProject updates an existing project's files
func (m *Manager) UpdateProject(config ProjectConfig) error {
	config.Override = true // Force override for updates
	return m.CreateProject(config)
}

// DeleteProject removes a project directory
func (m *Manager) DeleteProject(projectName string) error {
	if projectName == "" {
		return fmt.Errorf("project name is required")
	}

	projectPath := filepath.Join(m.basePath, projectName)

	// Check if project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project %s does not exist", projectName)
	}

	// Remove project directory
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("failed to delete project %s: %w", projectName, err)
	}

	return nil
}

// ListProjects returns a list of all compose projects
func (m *Manager) ListProjects() ([]map[string]interface{}, error) {
	// Read directory entries
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	projects := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directories
		}

		projectName := entry.Name()
		projectPath := filepath.Join(m.basePath, projectName)

		// Get file info for timestamps
		info, err := os.Stat(projectPath)
		if err != nil {
			continue // Skip if can't get info
		}

		// Look for compose file
		composeFilePath := filepath.Join(projectPath, "docker-compose.yml")
		if _, err := os.Stat(composeFilePath); os.IsNotExist(err) {
			// Try alternate filename
			composeFilePath = filepath.Join(projectPath, "compose.yml")
			if _, err := os.Stat(composeFilePath); os.IsNotExist(err) {
				continue // Skip if no compose file
			}
		}

		// Read compose content
		composeContent, _ := os.ReadFile(composeFilePath)

		// Check for .env file
		envContent := ""
		envFilePath := filepath.Join(projectPath, ".env")
		if envBytes, err := os.ReadFile(envFilePath); err == nil {
			envContent = string(envBytes)
		}

		// Format timestamps in RFC3339
		createdAt := info.ModTime().UTC().Format(time.RFC3339)
		updatedAt := createdAt

		project := map[string]interface{}{
			"id":             projectName,
			"name":           projectName,
			"path":           projectPath,
			"dirName":        projectName,
			"createdAt":      createdAt,
			"updatedAt":      updatedAt,
			"composeContent": string(composeContent),
			"envContent":     envContent,
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// ProjectExists checks if a project directory exists
func (m *Manager) ProjectExists(projectName string) bool {
	projectPath := filepath.Join(m.basePath, projectName)
	_, err := os.Stat(projectPath)
	return !os.IsNotExist(err)
}

// GetProjectPath returns the full path to a project directory
func (m *Manager) GetProjectPath(projectName string) string {
	return filepath.Join(m.basePath, projectName)
}

// GetComposePath returns the full path to a project's compose file
func (m *Manager) GetComposePath(projectName, composeFile string) string {
	if composeFile == "" {
		composeFile = "docker-compose.yml"
	}
	return filepath.Join(m.basePath, projectName, composeFile)
}

// writeFileIfNotExists writes content to a file, optionally overriding existing files
func (m *Manager) writeFileIfNotExists(filePath, content string, override bool) error {
	// Check if file exists
	if _, err := os.Stat(filePath); err == nil && !override {
		return fmt.Errorf("file %s already exists and override is false", filePath)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// generateEnvContent creates .env file content from environment variables
func (m *Manager) generateEnvContent(envVars map[string]string) string {
	content := "# Environment variables for Docker Compose\n"
	content += "# Generated by Arcane Agent\n\n"

	for key, value := range envVars {
		content += fmt.Sprintf("%s=%s\n", key, value)
	}

	return content
}
