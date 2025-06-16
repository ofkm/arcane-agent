package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
)

type ImageHandler struct {
	dockerClient *docker.Client
}

func NewImageHandler(dockerClient *docker.Client) *ImageHandler {
	return &ImageHandler{
		dockerClient: dockerClient,
	}
}

func (h *ImageHandler) ListImages(c *gin.Context) {
	allQuery := c.DefaultQuery("all", "false")
	all := allQuery == "true"

	images, err := h.dockerClient.ListImages(c.Request.Context(), all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"total":  len(images),
	})
}

func (h *ImageHandler) GetImage(c *gin.Context) {
	imageID := c.Param("id")
	image, err := h.dockerClient.GetImage(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, image)
}

func (h *ImageHandler) CreateImage(c *gin.Context) {
	var req struct {
		FromImage string `json:"fromImage" binding:"required"`
		Tag       string `json:"tag"`
		Platform  string `json:"platform"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default tag if not provided
	if req.Tag == "" {
		req.Tag = "latest"
	}

	err := h.dockerClient.PullImage(c.Request.Context(), req.FromImage, req.Tag, req.Platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image pulled successfully",
		"image":   req.FromImage + ":" + req.Tag,
	})
}

func (h *ImageHandler) DeleteImage(c *gin.Context) {
	imageID := c.Param("id")

	var req struct {
		Force   bool `json:"force"`
		NoPrune bool `json:"noPrune"`
	}

	// Bind query parameters or JSON body
	c.ShouldBindJSON(&req)
	if c.Query("force") == "true" {
		req.Force = true
	}
	if c.Query("noPrune") == "true" {
		req.NoPrune = true
	}

	deletedImages, err := h.dockerClient.RemoveImage(c.Request.Context(), imageID, req.Force, req.NoPrune)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Image deleted successfully",
		"deleted_images": deletedImages,
	})
}

func (h *ImageHandler) TagImage(c *gin.Context) {
	imageID := c.Param("id")

	var req struct {
		Repository string `json:"repository" binding:"required"`
		Tag        string `json:"tag"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default tag if not provided
	if req.Tag == "" {
		req.Tag = "latest"
	}

	err := h.dockerClient.TagImage(c.Request.Context(), imageID, req.Repository, req.Tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image tagged successfully",
		"source":  imageID,
		"target":  req.Repository + ":" + req.Tag,
	})
}

func (h *ImageHandler) PushImage(c *gin.Context) {
	imageID := c.Param("id")

	var req struct {
		Tag string `json:"tag"`
	}

	c.ShouldBindJSON(&req)

	err := h.dockerClient.PushImage(c.Request.Context(), imageID, req.Tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pushTarget := imageID
	if req.Tag != "" {
		pushTarget = imageID + ":" + req.Tag
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image pushed successfully",
		"image":   pushTarget,
	})
}
