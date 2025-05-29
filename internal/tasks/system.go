package tasks

import (
	"context"
	"os/exec"
	"runtime"
)

// SystemTaskExecutor handles system-level tasks
type SystemTaskExecutor struct{}

func NewSystemTaskExecutor() *SystemTaskExecutor {
	return &SystemTaskExecutor{}
}

func (s *SystemTaskExecutor) GetSystemInfo(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"platform":     runtime.GOOS,
		"architecture": runtime.GOARCH,
		"go_version":   runtime.Version(),
		"num_cpu":      runtime.NumCPU(),
	}, nil
}

func (s *SystemTaskExecutor) ExecuteCommand(ctx context.Context, command string, args []string) (interface{}, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"command": command,
		"args":    args,
		"output":  string(output),
	}, nil
}

func (s *SystemTaskExecutor) GetDiskUsage(ctx context.Context) (interface{}, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("wmic", "logicaldisk", "get", "size,freespace,caption")
	case "darwin":
		cmd = exec.Command("df", "-h")
	default: // linux
		cmd = exec.Command("df", "-h")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"disk_usage": string(output),
		"platform":   runtime.GOOS,
	}, nil
}

func (s *SystemTaskExecutor) GetMemoryUsage(ctx context.Context) (interface{}, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory")
	case "darwin":
		cmd = exec.Command("vm_stat")
	default: // linux
		cmd = exec.Command("free", "-h")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"memory_usage": string(output),
		"platform":     runtime.GOOS,
	}, nil
}
