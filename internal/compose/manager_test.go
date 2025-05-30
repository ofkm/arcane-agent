package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("/tmp/test-compose")

	if manager == nil {
		t.Error("Expected non-nil manager")
	}

	if manager.basePath != "/tmp/test-compose" {
		t.Errorf("Expected basePath '/tmp/test-compose', got '%s'", manager.basePath)
	}
}

func TestEnsureBaseDirectory(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "arcane-test-compose")
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)

	err := manager.EnsureBaseDirectory()
	if err != nil {
		t.Errorf("EnsureBaseDirectory failed: %v", err)
	}

	// Check if directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Base directory was not created")
	}
}

func TestCreateProject(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "arcane-test-compose")
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	manager.EnsureBaseDirectory()

	config := ProjectConfig{
		Name:    "test-project",
		Content: "version: '3.8'\nservices:\n  web:\n    image: nginx",
		EnvVars: map[string]string{
			"ENV":  "test",
			"PORT": "8080",
		},
	}

	err := manager.CreateProject(config)
	if err != nil {
		t.Errorf("CreateProject failed: %v", err)
	}

	// Check if project directory was created
	projectPath := filepath.Join(tempDir, "test-project")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Project directory was not created")
	}

	// Check if compose file was created
	composeFile := filepath.Join(projectPath, "docker-compose.yml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Error("Compose file was not created")
	}

	// Check if .env file was created
	envFile := filepath.Join(projectPath, ".env")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Error(".env file was not created")
	}

	// Check .env file content
	envContent, err := os.ReadFile(envFile)
	if err != nil {
		t.Errorf("Failed to read .env file: %v", err)
	}

	envStr := string(envContent)
	if !contains(envStr, "ENV=test") || !contains(envStr, "PORT=8080") {
		t.Errorf("Unexpected .env content: %s", envStr)
	}
}

func TestCreateProjectWithCustomComposeFile(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "arcane-test-compose")
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	manager.EnsureBaseDirectory()

	config := ProjectConfig{
		Name:        "test-project",
		ComposeFile: "docker-compose.prod.yml",
		Content:     "version: '3.8'\nservices:\n  web:\n    image: nginx",
	}

	err := manager.CreateProject(config)
	if err != nil {
		t.Errorf("CreateProject failed: %v", err)
	}

	// Check if custom compose file was created
	composeFile := filepath.Join(tempDir, "test-project", "docker-compose.prod.yml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Error("Custom compose file was not created")
	}
}

func TestCreateProjectValidation(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "arcane-test-compose")
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)

	tests := []struct {
		name    string
		config  ProjectConfig
		wantErr bool
	}{
		{
			name: "missing project name",
			config: ProjectConfig{
				Content: "version: '3.8'",
			},
			wantErr: true,
		},
		{
			name: "missing content",
			config: ProjectConfig{
				Name: "test-project",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: ProjectConfig{
				Name:    "test-project",
				Content: "version: '3.8'",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CreateProject(tt.config)
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "arcane-test-compose")
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	manager.EnsureBaseDirectory()

	// Create test projects
	projects := []string{"project1", "project2", "project3"}
	for _, name := range projects {
		config := ProjectConfig{
			Name:    name,
			Content: "version: '3.8'",
		}
		manager.CreateProject(config)
	}

	// List projects
	listedProjects, err := manager.ListProjects()
	if err != nil {
		t.Errorf("ListProjects failed: %v", err)
	}

	if len(listedProjects) != len(projects) {
		t.Errorf("Expected %d projects, got %d", len(projects), len(listedProjects))
	}

	// Check each project exists in the list
	for _, expected := range projects {
		found := false
		for _, actual := range listedProjects {
			if actual["name"] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Project %s not found in list", expected)
		}
	}
}

func TestDeleteProject(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "arcane-test-compose")
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	manager.EnsureBaseDirectory()

	// Create project
	config := ProjectConfig{
		Name:    "test-project",
		Content: "version: '3.8'",
	}
	manager.CreateProject(config)

	// Verify project exists
	if !manager.ProjectExists("test-project") {
		t.Error("Project should exist before deletion")
	}

	// Delete project
	err := manager.DeleteProject("test-project")
	if err != nil {
		t.Errorf("DeleteProject failed: %v", err)
	}

	// Verify project no longer exists
	if manager.ProjectExists("test-project") {
		t.Error("Project should not exist after deletion")
	}
}

func TestProjectExists(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "arcane-test-compose")
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	manager.EnsureBaseDirectory()

	// Check non-existent project
	if manager.ProjectExists("nonexistent") {
		t.Error("Non-existent project should not exist")
	}

	// Create project
	config := ProjectConfig{
		Name:    "test-project",
		Content: "version: '3.8'",
	}
	manager.CreateProject(config)

	// Check existing project
	if !manager.ProjectExists("test-project") {
		t.Error("Created project should exist")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}
