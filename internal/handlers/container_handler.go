package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
)

type ContainerHandler struct {
	dockerClient *docker.Client
}

func NewContainerHandler(dockerClient *docker.Client) *ContainerHandler {
	return &ContainerHandler{
		dockerClient: dockerClient,
	}
}

func (h *ContainerHandler) ListContainers(c *gin.Context) {
	allQuery := c.DefaultQuery("all", "true")
	all, _ := strconv.ParseBool(allQuery)

	containerList, err := h.dockerClient.ListContainers(c.Request.Context(), all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"containers": containerList,
		"total":      len(containerList),
	})
}

func (h *ContainerHandler) GetContainer(c *gin.Context) {
	containerID := c.Param("id")
	container, err := h.dockerClient.GetContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, container)
}

func (h *ContainerHandler) StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	err := h.dockerClient.StartContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Container started successfully",
		"container_id": containerID,
	})
}

func (h *ContainerHandler) StopContainer(c *gin.Context) {
	containerID := c.Param("id")
	err := h.dockerClient.StopContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Container stopped successfully",
		"container_id": containerID,
	})
}

func (h *ContainerHandler) RestartContainer(c *gin.Context) {
	containerID := c.Param("id")
	err := h.dockerClient.RestartContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Container restarted successfully",
		"container_id": containerID,
	})
}
