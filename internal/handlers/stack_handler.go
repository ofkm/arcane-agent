package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/dto"
	"github.com/ofkm/arcane-agent/internal/models"
	"github.com/ofkm/arcane-agent/internal/services"
)

type StackHandler struct {
	stackService *services.StackService
}

func NewStackHandler(stackService *services.StackService) *StackHandler {
	return &StackHandler{
		stackService: stackService,
	}
}

func (h *StackHandler) ListStacks(c *gin.Context) {
	stacks, err := h.stackService.ListStacks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to fetch stacks: %v", err),
		})
		return
	}

	var stackList []map[string]interface{}
	for _, stack := range stacks {
		services, err := h.stackService.GetStackServices(c.Request.Context(), stack.ID)
		var serviceCount, runningCount int
		if err != nil {
			fmt.Printf("Warning: failed to get services for stack %s: %v\n", stack.ID, err)
			serviceCount = stack.ServiceCount
			runningCount = stack.RunningCount
			services = nil
		} else {
			serviceCount = len(services)
			runningCount = 0
			for _, service := range services {
				if service.Status == "running" {
					runningCount++
				}
			}
		}

		stackResponse := map[string]interface{}{
			"id":           stack.ID,
			"name":         stack.Name,
			"path":         stack.Path,
			"status":       stack.Status,
			"serviceCount": serviceCount,
			"runningCount": runningCount,
			"createdAt":    stack.CreatedAt,
			"updatedAt":    stack.UpdatedAt,
			"autoUpdate":   stack.AutoUpdate,
			"isExternal":   stack.IsExternal,
			"isLegacy":     stack.IsLegacy,
			"isRemote":     stack.IsRemote,
			"services":     services,
		}
		stackList = append(stackList, stackResponse)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stacks":  stackList,
		"count":   len(stackList),
	})
}

func (h *StackHandler) CreateStack(c *gin.Context) {
	var req dto.CreateStackDto
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	createdStack, err := h.stackService.CreateStack(
		c.Request.Context(),
		req.Name,
		req.ComposeContent,
		req.EnvContent,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create stack",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"stack":   createdStack,
	})
}

func (h *StackHandler) GetStack(c *gin.Context) {
	stackID := c.Param("id")

	stack, err := h.stackService.GetStackByID(c.Request.Context(), stackID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Stack not found",
		})
		return
	}

	composeContent, envContent, err := h.stackService.GetStackContent(c.Request.Context(), stackID)
	if err != nil {
		fmt.Printf("Warning: failed to read stack content: %v\n", err)
		composeContent, envContent = "", ""
	}

	services, err := h.stackService.GetStackServices(c.Request.Context(), stackID)
	if err != nil {
		fmt.Printf("Warning: failed to get services: %v\n", err)
		services = nil
	}

	var serviceCount, runningCount int
	if services != nil {
		serviceCount = len(services)
		for _, service := range services {
			if service.Status == "running" || service.Status == "Up" {
				runningCount++
			}
		}
	} else {
		serviceCount = stack.ServiceCount
		runningCount = stack.RunningCount
	}

	stackResponse := map[string]interface{}{
		"id":             stack.ID,
		"name":           stack.Name,
		"path":           stack.Path,
		"composeContent": composeContent,
		"envContent":     envContent,
		"status":         stack.Status,
		"serviceCount":   serviceCount,
		"runningCount":   runningCount,
		"createdAt":      stack.CreatedAt,
		"updatedAt":      stack.UpdatedAt,
		"autoUpdate":     stack.AutoUpdate,
		"isExternal":     stack.IsExternal,
		"isLegacy":       stack.IsLegacy,
		"isRemote":       stack.IsRemote,
		"services":       services,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stack":   stackResponse,
	})
}

func (h *StackHandler) UpdateStack(c *gin.Context) {
	stackID := c.Param("id")

	var req dto.UpdateStackDto
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if req.ComposeContent != nil || req.EnvContent != nil {
		if err := h.stackService.UpdateStackContent(c.Request.Context(), stackID, req.ComposeContent, req.EnvContent); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to update stack content",
			})
			return
		}
	}

	if req.Name != nil || req.AutoUpdate != nil {
		stack, err := h.stackService.GetStackByID(c.Request.Context(), stackID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Stack not found",
			})
			return
		}

		if req.Name != nil {
			stack.Name = *req.Name
		}
		if req.AutoUpdate != nil {
			stack.AutoUpdate = *req.AutoUpdate
		}

		if _, err := h.stackService.UpdateStack(c.Request.Context(), stack); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to update stack",
			})
			return
		}
	}

	updatedStack, err := h.stackService.GetStackByID(c.Request.Context(), stackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get updated stack",
		})
		return
	}

	services, err := h.stackService.GetStackServices(c.Request.Context(), stackID)
	if err != nil {
		fmt.Printf("Warning: failed to get services: %v\n", err)
		services = nil
	}

	stackResponse := map[string]interface{}{
		"id":           updatedStack.ID,
		"name":         updatedStack.Name,
		"path":         updatedStack.Path,
		"status":       updatedStack.Status,
		"serviceCount": len(services),
		"runningCount": updatedStack.RunningCount,
		"createdAt":    updatedStack.CreatedAt,
		"updatedAt":    updatedStack.UpdatedAt,
		"autoUpdate":   updatedStack.AutoUpdate,
		"isExternal":   updatedStack.IsExternal,
		"isLegacy":     updatedStack.IsLegacy,
		"isRemote":     updatedStack.IsRemote,
		"services":     services,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stack":   stackResponse,
	})
}

func (h *StackHandler) DeleteStack(c *gin.Context) {
	stackID := c.Param("id")

	err := h.stackService.DeleteStack(c.Request.Context(), stackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete stack",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack deleted successfully",
	})
}

func (h *StackHandler) StartStack(c *gin.Context) {
	stackID := c.Param("id")

	err := h.stackService.DeployStack(c.Request.Context(), stackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to start stack",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack started successfully",
	})
}

func (h *StackHandler) StopStack(c *gin.Context) {
	stackID := c.Param("id")

	if err := h.stackService.StopStack(c.Request.Context(), stackID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to stop stack",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack stopped successfully",
	})
}

func (h *StackHandler) RestartStack(c *gin.Context) {
	stackID := c.Param("id")

	if err := h.stackService.RestartStack(c.Request.Context(), stackID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to restart stack",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack restarted successfully",
		"stackId": stackID,
	})
}

func (h *StackHandler) RedeployStack(c *gin.Context) {
	stackID := c.Param("id")

	var req dto.RedeployStackDto
	if err := c.ShouldBindJSON(&req); err != nil {
		req = dto.RedeployStackDto{
			Profiles:     []string{},
			EnvOverrides: map[string]string{},
		}
	}

	if err := h.stackService.RedeployStack(c.Request.Context(), stackID, req.Profiles, req.EnvOverrides); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to redeploy stack: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack redeployed successfully",
	})
}

func (h *StackHandler) DownStack(c *gin.Context) {
	stackID := c.Param("id")

	if err := h.stackService.DownStack(c.Request.Context(), stackID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to bring down stack: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack brought down successfully",
	})
}

func (h *StackHandler) DestroyStack(c *gin.Context) {
	stackID := c.Param("id")

	var req dto.DestroyStackDto
	if err := c.ShouldBindJSON(&req); err != nil {
		req = dto.DestroyStackDto{
			RemoveFiles:   false,
			RemoveVolumes: false,
		}
	}

	if err := h.stackService.DestroyStack(c.Request.Context(), stackID, req.RemoveFiles, req.RemoveVolumes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to destroy stack: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack destroyed successfully",
	})
}

func (h *StackHandler) PullStack(c *gin.Context) {
	stackID := c.Param("id")

	if err := h.stackService.PullStackImages(c.Request.Context(), stackID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to pull stack images",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack images pulled successfully",
		"stackId": stackID,
	})
}

func (h *StackHandler) DeployStack(c *gin.Context) {
	stackID := c.Param("id")

	var req struct {
		Profiles      []string          `json:"profiles"`
		EnvOverrides  map[string]string `json:"env_overrides"`
		ForceRecreate bool              `json:"force_recreate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := h.stackService.DeployStack(c.Request.Context(), stackID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Stack deployed successfully",
	})
}

func (h *StackHandler) GetStackServices(c *gin.Context) {
	stackID := c.Param("id")

	services, err := h.stackService.GetStackServices(c.Request.Context(), stackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    services,
	})
}

func (h *StackHandler) PullImages(c *gin.Context) {
	stackID := c.Param("id")

	if err := h.stackService.PullStackImages(c.Request.Context(), stackID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Images pulled successfully",
	})
}

func (h *StackHandler) ConvertDockerRun(c *gin.Context) {
	var req models.ConvertDockerRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// For now, return a simple conversion - you can implement a full converter later
	c.JSON(http.StatusOK, models.ConvertDockerRunResponse{
		Success:       true,
		DockerCompose: "# Docker Compose conversion not implemented in agent yet",
		EnvVars:       map[string]string{},
		ServiceName:   "app",
	})
}

func (h *StackHandler) GetStackLogsStream(c *gin.Context) {
	stackID := c.Param("id")
	if stackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Stack ID is required",
		})
		return
	}

	// Get query parameters for log options
	follow := c.DefaultQuery("follow", "true") == "true"
	tail := c.DefaultQuery("tail", "100")
	since := c.Query("since")
	timestamps := c.DefaultQuery("timestamps", "true") == "true"

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	logsChan := make(chan string, 100)
	errChan := make(chan error, 1)

	// Start streaming logs in a goroutine
	go func() {
		defer close(logsChan)
		defer close(errChan)

		err := h.stackService.StreamStackLogs(c.Request.Context(), stackID, logsChan, follow, tail, since, timestamps)
		if err != nil {
			errChan <- err
		}
	}()

	// Send logs to client
	c.Stream(func(w io.Writer) bool {
		select {
		case logLine, ok := <-logsChan:
			if !ok {
				return false
			}

			logData := h.parseStackLogLine(logLine)
			c.SSEvent("log", logData)
			return true

		case err := <-errChan:
			c.SSEvent("error", gin.H{"error": err.Error()})
			return false

		case <-c.Request.Context().Done():
			return false
		}
	})
}

func (h *StackHandler) parseStackLogLine(logLine string) gin.H {
	var service, message, timestamp string
	var level = "info"

	if strings.HasPrefix(logLine, "[STDERR] ") {
		level = "stderr"
		logLine = strings.TrimPrefix(logLine, "[STDERR] ")
	}

	parts := strings.SplitN(logLine, " ", 2)
	if len(parts) == 2 && strings.Contains(parts[0], "T") && strings.Contains(parts[0], "Z") {
		timestamp = parts[0]
		logLine = parts[1]
	} else {
		timestamp = time.Now().Format(time.RFC3339Nano)
	}

	if strings.Contains(logLine, " | ") {
		serviceParts := strings.SplitN(logLine, " | ", 2)
		if len(serviceParts) == 2 {
			service = strings.TrimSpace(serviceParts[0])
			message = serviceParts[1]
		} else {
			message = logLine
		}
	} else {
		message = logLine
	}

	return gin.H{
		"level":     level,
		"message":   message,
		"timestamp": timestamp,
		"service":   service,
	}
}
