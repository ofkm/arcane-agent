package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
	"github.com/ofkm/arcane-agent/internal/dto"
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"images": images,
			"total":  len(images),
		},
		"success": true,
	})
}

func (h *ImageHandler) GetImage(c *gin.Context) {
	imageID := c.Param("id")
	image, err := h.dockerClient.GetImage(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    image,
		"success": true,
	})
}

func (h *ImageHandler) Pull(c *gin.Context) {
	var req dto.ImagePullDto

	// Read the raw body to log it
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("Failed to read request body", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to read request body",
		})
		return
	}

	// Restore the body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Add debug logging
	slog.Info("Pull request received",
		"method", c.Request.Method,
		"contentType", c.GetHeader("Content-Type"),
		"bodyContent", string(bodyBytes))

	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("Failed to bind JSON", "error", err.Error(), "rawBody", string(bodyBytes))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	slog.Info("Pull request parsed", "imageName", req.ImageName)

	c.Writer.Header().Set("Content-Type", "application/x-json-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	err = h.dockerClient.PullImageWithStream(c.Request.Context(), req.ImageName, c.Writer)

	if err != nil {
		if !c.Writer.Written() {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "manifest unknown") {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"error":   fmt.Sprintf("Failed to pull image '%s': %s. Ensure the image name and tag are correct and the image exists in the registry.", req.ImageName, err.Error()),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   fmt.Sprintf("Failed to pull image '%s': %s", req.ImageName, err.Error()),
				})
			}
		} else {
			slog.Error("Error during image pull stream or post-stream operation", "imageName", req.ImageName, "error", err.Error())
			fmt.Fprintf(c.Writer, `{"error": {"code": 500, "message": "Stream interrupted or post-stream operation failed: %s"}}`+"\n", strings.ReplaceAll(err.Error(), "\"", "'"))
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		return
	}

	slog.Info("Image pull stream completed", "imageName", req.ImageName)
}

// Keep the existing CreateImage method for backward compatibility
func (h *ImageHandler) CreateImage(c *gin.Context) {
	var req struct {
		FromImage string `json:"fromImage" binding:"required"`
		Tag       string `json:"tag"`
		Platform  string `json:"platform"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set default tag if not provided
	if req.Tag == "" {
		req.Tag = "latest"
	}

	err := h.dockerClient.PullImage(c.Request.Context(), req.FromImage, req.Tag, req.Platform)
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
			"message": "Image pulled successfully",
			"image":   req.FromImage + ":" + req.Tag,
		},
		"success": true,
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message":        "Image deleted successfully",
			"deleted_images": deletedImages,
		},
		"success": true,
	})
}

func (h *ImageHandler) TagImage(c *gin.Context) {
	imageID := c.Param("id")

	var req struct {
		Repository string `json:"repository" binding:"required"`
		Tag        string `json:"tag"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set default tag if not provided
	if req.Tag == "" {
		req.Tag = "latest"
	}

	err := h.dockerClient.TagImage(c.Request.Context(), imageID, req.Repository, req.Tag)
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
			"message": "Image tagged successfully",
			"source":  imageID,
			"target":  req.Repository + ":" + req.Tag,
		},
		"success": true,
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	pushTarget := imageID
	if req.Tag != "" {
		pushTarget = imageID + ":" + req.Tag
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"message": "Image pushed successfully",
			"image":   pushTarget,
		},
		"success": true,
	})
}
