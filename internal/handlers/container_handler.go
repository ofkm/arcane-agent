package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
	"github.com/ofkm/arcane-agent/internal/services"
)

type ContainerHandler struct {
	dockerClient     *docker.Client
	containerService *services.ContainerService
}

func NewContainerHandler(dockerClient *docker.Client) *ContainerHandler {
	return &ContainerHandler{
		dockerClient:     dockerClient,
		containerService: services.NewContainerService(dockerClient),
	}
}

func (h *ContainerHandler) ListContainers(c *gin.Context) {
	allQuery := c.DefaultQuery("all", "true")
	all, _ := strconv.ParseBool(allQuery)

	containerList, err := h.dockerClient.ListContainers(c.Request.Context(), all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"containers": containerList,
			"total":      len(containerList),
		},
		"success": true,
	})
}

func (h *ContainerHandler) GetContainer(c *gin.Context) {
	containerID := c.Param("id")
	container, err := h.dockerClient.GetContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    container,
		"success": true,
	})
}

func (h *ContainerHandler) StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	err := h.dockerClient.StartContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message":      "Container started successfully",
			"container_id": containerID,
		},
		"success": true,
	})
}

func (h *ContainerHandler) StopContainer(c *gin.Context) {
	containerID := c.Param("id")
	err := h.dockerClient.StopContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message":      "Container stopped successfully",
			"container_id": containerID,
		},
		"success": true,
	})
}

func (h *ContainerHandler) RestartContainer(c *gin.Context) {
	containerID := c.Param("id")
	err := h.dockerClient.RestartContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message":      "Container restarted successfully",
			"container_id": containerID,
		},
		"success": true,
	})
}

// GetStats returns container resource usage statistics
func (h *ContainerHandler) GetStats(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":    nil,
			"success": false,
			"error":   "Container ID is required",
		})
		return
	}

	// Check if streaming is requested
	stream := c.Query("stream") == "true"

	stats, err := h.containerService.GetStats(c.Request.Context(), containerID, stream)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    stats,
		"success": true,
	})
}

func (h *ContainerHandler) GetStatsStream(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":    nil,
			"success": false,
			"error":   "Container ID is required",
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	statsChan := make(chan interface{}, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(statsChan)
		defer close(errChan)

		err := h.containerService.StreamStats(c.Request.Context(), containerID, statsChan)
		if err != nil {
			errChan <- err
		}
	}()

	// Send stats to client
	c.Stream(func(w io.Writer) bool {
		select {
		case stats, ok := <-statsChan:
			if !ok {
				return false
			}
			c.SSEvent("stats", stats)
			return true
		case err := <-errChan:
			c.SSEvent("error", gin.H{"error": err.Error()})
			return false
		case <-c.Request.Context().Done():
			return false
		}
	})
}
