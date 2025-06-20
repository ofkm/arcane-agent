package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/google/uuid"
	"github.com/ofkm/arcane-agent/internal/models"
)

type StackService struct {
	stacksDir string
}

func NewStackService() *StackService {
	return &StackService{
		stacksDir: "data/stacks",
	}
}

type StackInfo struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Status       string                    `json:"status"`
	Services     []models.StackServiceInfo `json:"services"`
	ServiceCount int                       `json:"service_count"`
	RunningCount int                       `json:"running_count"`
	ComposeYAML  string                    `json:"compose_yaml,omitempty"`
}

func (s *StackService) CreateStack(ctx context.Context, name, composeContent string, envContent *string) (*models.Stack, error) {
	stackID := uuid.New().String()
	folderName := s.sanitizeStackName(name)

	stackPath := filepath.Join(s.stacksDir, folderName)

	counter := 1
	originalPath := stackPath
	for {
		if _, err := os.Stat(stackPath); os.IsNotExist(err) {
			break
		}
		stackPath = fmt.Sprintf("%s-%d", originalPath, counter)
		folderName = fmt.Sprintf("%s-%d", s.sanitizeStackName(name), counter)
		counter++
	}

	stack := &models.Stack{
		ID:           stackID,
		Name:         name,
		DirName:      &folderName,
		Path:         stackPath,
		Status:       models.StackStatusStopped,
		ServiceCount: 0,
		RunningCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.saveStackFiles(stackPath, composeContent, envContent); err != nil {
		return nil, fmt.Errorf("failed to save stack files: %w", err)
	}

	return stack, nil
}

func (s *StackService) DeployStack(ctx context.Context, stackID string) error {
	// Find stack directly by ID without calling ListStacks
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return fmt.Errorf("stack not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker-compose", "up", "-d")
	cmd.Dir = stack.Path

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy stack: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (s *StackService) StopStack(ctx context.Context, stackID string) error {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return fmt.Errorf("stack not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker-compose", "stop")
	cmd.Dir = stack.Path
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop stack: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (s *StackService) DownStack(ctx context.Context, stackID string) error {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return fmt.Errorf("stack not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker-compose", "down")
	cmd.Dir = stack.Path
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to down stack: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (s *StackService) RestartStack(ctx context.Context, stackID string) error {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return fmt.Errorf("stack not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker-compose", "restart")
	cmd.Dir = stack.Path
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart stack: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (s *StackService) PullStackImages(ctx context.Context, stackID string) error {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return fmt.Errorf("stack not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker-compose", "pull")
	cmd.Dir = stack.Path
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull stack images: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (s *StackService) RedeployStack(ctx context.Context, stackID string, profiles []string, envOverrides map[string]string) error {
	if err := s.PullStackImages(ctx, stackID); err != nil {
		fmt.Printf("Warning: failed to pull images: %v\n", err)
	}

	if err := s.StopStack(ctx, stackID); err != nil {
		return fmt.Errorf("failed to stop stack for redeploy: %w", err)
	}

	return s.DeployStack(ctx, stackID)
}

func (s *StackService) DestroyStack(ctx context.Context, stackID string, removeFiles, removeVolumes bool) error {
	stacks, err := s.ListStacks(ctx)
	if err != nil {
		return err
	}

	var stack *models.Stack
	for _, st := range stacks {
		if st.ID == stackID {
			stack = &st
			break
		}
	}

	if stack == nil {
		return fmt.Errorf("stack not found")
	}

	if err := s.DownStack(ctx, stackID); err != nil {
		fmt.Printf("Warning: failed to bring down stack: %v\n", err)
	}

	if removeVolumes {
		cmd := exec.CommandContext(ctx, "docker-compose", "down", "-v")
		cmd.Dir = stack.Path
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Warning: failed to remove volumes: %v\nOutput: %s\n", err, string(output))
		}
	}

	if removeFiles {
		if err := os.RemoveAll(stack.Path); err != nil {
			return fmt.Errorf("failed to remove stack files: %w", err)
		}
	}

	return nil
}

func (s *StackService) ListStacks(ctx context.Context) ([]models.Stack, error) {
	var stacks []models.Stack

	if _, err := os.Stat(s.stacksDir); os.IsNotExist(err) {
		return stacks, nil
	}

	entries, err := os.ReadDir(s.stacksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read stacks directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		stackPath := filepath.Join(s.stacksDir, entry.Name())
		folderName := fmt.Sprintf("%s-%d", s.sanitizeStackName(entry.Name()))
		composeFile := s.findComposeFile(stackPath)
		if composeFile == "" {
			continue
		}

		// Read stack metadata if exists
		metadataPath := filepath.Join(stackPath, ".stack-metadata.json")
		stack := models.Stack{
			ID:           entry.Name(), // Use directory name as ID for now
			Name:         entry.Name(),
			DirName:      &folderName,
			Path:         stackPath,
			Status:       models.StackStatusUnknown,
			ServiceCount: 0,
			RunningCount: 0,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Try to read metadata
		if metadataBytes, err := os.ReadFile(metadataPath); err == nil {
			var metadata struct {
				ID        string    `json:"id"`
				Name      string    `json:"name"`
				CreatedAt time.Time `json:"createdAt"`
			}
			if err := json.Unmarshal(metadataBytes, &metadata); err == nil {
				stack.ID = metadata.ID
				stack.Name = metadata.Name
				stack.CreatedAt = metadata.CreatedAt
			}
		}

		services, err := s.getStackServicesDirectly(ctx, &stack)
		if err == nil {
			stack.ServiceCount = len(services)
			runningCount := 0
			for _, service := range services {
				if service.Status == "running" || service.Status == "Up" {
					runningCount++
				}
			}
			stack.RunningCount = runningCount

			if stack.ServiceCount == 0 {
				stack.Status = models.StackStatusStopped
			} else if runningCount == stack.ServiceCount {
				stack.Status = models.StackStatusRunning
			} else if runningCount > 0 {
				stack.Status = models.StackStatusPartiallyRunning
			} else {
				stack.Status = models.StackStatusStopped
			}
		}

		stacks = append(stacks, stack)
	}

	return stacks, nil
}

// Add this helper method to avoid recursion
func (s *StackService) getStackServicesDirectly(ctx context.Context, stack *models.Stack) ([]models.StackServiceInfo, error) {
	cmd := exec.CommandContext(ctx, "docker-compose", "ps", "--format", "json")
	cmd.Dir = stack.Path
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	var services []models.StackServiceInfo

	output, err := cmd.Output()
	if err == nil {
		services, err = s.parseComposePS(string(output))
		if err != nil {
			return nil, fmt.Errorf("failed to parse compose ps output: %w", err)
		}
	}

	if len(services) > 0 {
		return services, nil
	}

	composeFile := s.findComposeFile(stack.Path)
	if composeFile == "" {
		return []models.StackServiceInfo{}, nil
	}

	servicesFromFile, err := s.parseServicesFromComposeFile(composeFile, stack.Name)
	if err != nil {
		return []models.StackServiceInfo{}, nil
	}

	return servicesFromFile, nil
}

func (s *StackService) GetStackByID(ctx context.Context, id string) (*models.Stack, error) {
	stacks, err := s.ListStacks(ctx)
	if err != nil {
		return nil, err
	}

	for _, stack := range stacks {
		if stack.ID == id {
			return &stack, nil
		}
	}

	return nil, fmt.Errorf("stack not found")
}

func (s *StackService) UpdateStack(ctx context.Context, stack *models.Stack) (*models.Stack, error) {
	// Save metadata
	metadataPath := filepath.Join(stack.Path, ".stack-metadata.json")
	metadata := struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}{
		ID:        stack.ID,
		Name:      stack.Name,
		CreatedAt: stack.CreatedAt,
		UpdatedAt: time.Now(),
	}

	metadataBytes, _ := json.Marshal(metadata)
	os.WriteFile(metadataPath, metadataBytes, 0644)

	stack.UpdatedAt = time.Now()
	return stack, nil
}

func (s *StackService) UpdateStackContent(ctx context.Context, stackID string, composeContent, envContent *string) error {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return err
	}

	if composeContent != nil {
		existingComposeFile := s.findComposeFile(stack.Path)
		var composePath string

		if existingComposeFile != "" {
			composePath = existingComposeFile
		} else {
			composePath = filepath.Join(stack.Path, "compose.yaml")
		}

		if err := os.WriteFile(composePath, []byte(*composeContent), 0644); err != nil {
			return fmt.Errorf("failed to update compose file: %w", err)
		}
	}

	if envContent != nil {
		envPath := filepath.Join(stack.Path, ".env")
		if *envContent == "" {
			os.Remove(envPath)
		} else {
			if err := os.WriteFile(envPath, []byte(*envContent), 0644); err != nil {
				return fmt.Errorf("failed to update env file: %w", err)
			}
		}
	}

	return nil
}

func (s *StackService) GetStackContent(ctx context.Context, stackID string) (composeContent, envContent string, err error) {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return "", "", err
	}

	composeFile := s.findComposeFile(stack.Path)
	if composeFile != "" {
		if content, err := os.ReadFile(composeFile); err == nil {
			composeContent = string(content)
		}
	}

	envPath := filepath.Join(stack.Path, ".env")
	if content, err := os.ReadFile(envPath); err == nil {
		envContent = string(content)
	}

	return composeContent, envContent, nil
}

func (s *StackService) DeleteStack(ctx context.Context, stackID string) error {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return err
	}

	if stack.Status == models.StackStatusRunning {
		if err := s.DownStack(ctx, stackID); err != nil {
			fmt.Printf("Warning: failed to stop stack before deletion: %v\n", err)
		}
	}

	if err := os.RemoveAll(stack.Path); err != nil {
		fmt.Printf("Warning: failed to remove stack directory %s: %v\n", stack.Path, err)
	}

	return nil
}

func (s *StackService) GetStackServices(ctx context.Context, stackID string) ([]models.StackServiceInfo, error) {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "docker-compose", "ps", "--format", "json")
	cmd.Dir = stack.Path
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	var services []models.StackServiceInfo

	output, err := cmd.Output()
	if err == nil {
		services, err = s.parseComposePS(string(output))
		if err != nil {
			return nil, fmt.Errorf("failed to parse compose ps output: %w", err)
		}
	}

	if len(services) > 0 {
		return services, nil
	}

	composeFile := s.findComposeFile(stack.Path)
	if composeFile == "" {
		return []models.StackServiceInfo{}, nil
	}

	servicesFromFile, err := s.parseServicesFromComposeFile(composeFile, stack.Name)
	if err != nil {
		return []models.StackServiceInfo{}, nil
	}

	return servicesFromFile, nil
}

func (s *StackService) StreamStackLogs(ctx context.Context, stackID string, logsChan chan<- string, follow bool, tail, since string, timestamps bool) error {
	stack, err := s.GetStackByID(ctx, stackID)
	if err != nil {
		return err
	}

	args := []string{"logs"}
	if tail != "" {
		args = append(args, "--tail", tail)
	}
	if since != "" {
		args = append(args, "--since", since)
	}
	if timestamps {
		args = append(args, "--timestamps")
	}
	if follow {
		args = append(args, "--follow")
	}

	cmd := exec.CommandContext(ctx, "docker-compose", args...)
	cmd.Dir = stack.Path
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", stack.Name),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker-compose logs: %w", err)
	}

	// Handle stdout and stderr concurrently
	done := make(chan error, 2)

	// Read stdout
	go func() {
		done <- s.readStackLogsFromReader(ctx, stdout, logsChan, "stdout")
	}()

	// Read stderr
	go func() {
		done <- s.readStackLogsFromReader(ctx, stderr, logsChan, "stderr")
	}()

	// Wait for command completion or context cancellation
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return ctx.Err()
	case err := <-done:
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		if err != nil && err != io.EOF {
			return err
		}
		return nil
	}
}

func (s *StackService) readStackLogsFromReader(ctx context.Context, reader io.Reader, logsChan chan<- string, source string) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scanner.Text()
			if line != "" {
				if source == "stderr" {
					line = "[STDERR] " + line
				}

				select {
				case logsChan <- line:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	return scanner.Err()
}

// Helper methods
func (s *StackService) sanitizeStackName(name string) string {
	name = strings.TrimSpace(name)
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
}

func (s *StackService) saveStackFiles(stackPath, composeContent string, envContent *string) error {
	if err := os.MkdirAll(stackPath, 0755); err != nil {
		return fmt.Errorf("failed to create stack directory: %w", err)
	}

	// Save metadata
	stackID := uuid.New().String()
	metadata := struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
	}{
		ID:        stackID,
		Name:      filepath.Base(stackPath),
		CreatedAt: time.Now(),
	}

	metadataBytes, _ := json.Marshal(metadata)
	metadataPath := filepath.Join(stackPath, ".stack-metadata.json")
	os.WriteFile(metadataPath, metadataBytes, 0644)

	existingComposeFile := s.findComposeFile(stackPath)
	var composePath string

	if existingComposeFile != "" {
		composePath = existingComposeFile
	} else {
		composePath = filepath.Join(stackPath, "compose.yaml")
	}

	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to save compose file: %w", err)
	}

	if envContent != nil && *envContent != "" {
		envPath := filepath.Join(stackPath, ".env")
		if err := os.WriteFile(envPath, []byte(*envContent), 0644); err != nil {
			return fmt.Errorf("failed to save env file: %w", err)
		}
	}

	return nil
}

func (s *StackService) findComposeFile(stackDir string) string {
	possibleFiles := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yml",
		"docker-compose.yaml",
	}

	for _, filename := range possibleFiles {
		fullPath := filepath.Join(stackDir, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

func (s *StackService) parseComposePS(output string) ([]models.StackServiceInfo, error) {
	if strings.TrimSpace(output) == "" {
		return []models.StackServiceInfo{}, nil
	}

	var services []models.StackServiceInfo

	if strings.HasPrefix(strings.TrimSpace(output), "[") {
		var psOutput []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &psOutput); err == nil {
			for _, item := range psOutput {
				service := s.parseComposeService(item)
				if service != nil {
					services = append(services, *service)
				}
			}
			return services, nil
		}
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}

		service := s.parseComposeService(item)
		if service != nil {
			services = append(services, *service)
		}
	}

	return services, nil
}

func (s *StackService) parseComposeService(item map[string]interface{}) *models.StackServiceInfo {
	service := &models.StackServiceInfo{}

	if name, ok := item["Name"].(string); ok {
		service.Name = name
	} else if service_name, ok := item["Service"].(string); ok {
		service.Name = service_name
	}

	if image, ok := item["Image"].(string); ok {
		service.Image = image
	}

	if state, ok := item["State"].(string); ok {
		service.Status = state
	} else if status, ok := item["Status"].(string); ok {
		service.Status = status
	}

	if id, ok := item["ID"].(string); ok {
		service.ContainerID = id
	} else if container_id, ok := item["ContainerID"].(string); ok {
		service.ContainerID = container_id
	}

	if portsInterface, ok := item["Ports"]; ok {
		switch ports := portsInterface.(type) {
		case string:
			if ports != "" {
				service.Ports = []string{ports}
			}
		case []interface{}:
			for _, port := range ports {
				if portStr, ok := port.(string); ok && portStr != "" {
					service.Ports = append(service.Ports, portStr)
				}
			}
		case []string:
			service.Ports = ports
		}
	}

	if service.Name == "" {
		return nil
	}

	return service
}

func (s *StackService) parseServicesFromComposeFile(composeFile, stackName string) ([]models.StackServiceInfo, error) {
	options, err := cli.NewProjectOptions(
		[]string{composeFile},
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithName(stackName),
		cli.WithWorkingDirectory(filepath.Dir(composeFile)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project options: %w", err)
	}

	project, err := options.LoadProject(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	var services []models.StackServiceInfo

	for _, service := range project.Services {
		serviceInfo := models.StackServiceInfo{
			Name:        service.Name,
			Image:       service.Image,
			Status:      "not created",
			ContainerID: "",
			Ports:       []string{},
		}

		for _, port := range service.Ports {
			if port.Published != "" && port.Target != 0 {
				portStr := fmt.Sprintf("%s:%d", port.Published, port.Target)
				if port.Protocol != "" {
					portStr += "/" + port.Protocol
				}
				serviceInfo.Ports = append(serviceInfo.Ports, portStr)
			}
		}

		services = append(services, serviceInfo)
	}

	return services, nil
}
