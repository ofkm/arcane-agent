package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
)

type DockerHandler struct {
	dockerClient *docker.Client
}

func NewDockerHandler(dockerClient *docker.Client) *DockerHandler {
	return &DockerHandler{
		dockerClient: dockerClient,
	}
}

func (h *DockerHandler) GetDockerInfo(c *gin.Context) {
	info, err := h.dockerClient.GetSystemInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    info,
		"success": true,
	})
}
