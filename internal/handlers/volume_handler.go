package handlers

import (
	"net/http"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
)

type VolumeHandler struct {
	dockerClient *docker.Client
}

func NewVolumeHandler(dockerClient *docker.Client) *VolumeHandler {
	return &VolumeHandler{
		dockerClient: dockerClient,
	}
}

func (h *VolumeHandler) ListVolumes(c *gin.Context) {
	response, err := h.dockerClient.ListVolumes(c.Request.Context())
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
			"volumes": response.Volumes,
			"total":   len(response.Volumes),
		},
		"success": true,
	})
}

func (h *VolumeHandler) GetVolume(c *gin.Context) {
	volumeID := c.Param("id")
	volume, err := h.dockerClient.GetVolume(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    volume,
		"success": true,
	})
}

func (h *VolumeHandler) GetVolumeUsage(c *gin.Context) {
	volumeID := c.Param("id")
	inUse, usingContainers, err := h.dockerClient.GetVolumeUsage(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"inUse":      inUse,
			"containers": usingContainers,
		},
	})

}

func (h *VolumeHandler) CreateVolume(c *gin.Context) {
	var req struct {
		Name       string            `json:"name"`
		Driver     string            `json:"driver"`
		DriverOpts map[string]string `json:"driverOpts"`
		Labels     map[string]string `json:"labels"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set default driver
	if req.Driver == "" {
		req.Driver = "local"
	}

	options := volume.CreateOptions{
		Name:       req.Name,
		Driver:     req.Driver,
		DriverOpts: req.DriverOpts,
		Labels:     req.Labels,
	}

	volume, err := h.dockerClient.CreateVolume(c.Request.Context(), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"message":    "Volume created successfully",
			"volume_id":  volume.Name,
			"driver":     volume.Driver,
			"mountpoint": volume.Mountpoint,
		},
		"success": true,
	})
}

func (h *VolumeHandler) DeleteVolume(c *gin.Context) {
	volumeID := c.Param("id")

	var req struct {
		Force bool `json:"force"`
	}

	// Check for force parameter in query or body
	c.ShouldBindJSON(&req)
	if c.Query("force") == "true" {
		req.Force = true
	}

	err := h.dockerClient.RemoveVolume(c.Request.Context(), volumeID, req.Force)
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
			"message":   "Volume deleted successfully",
			"volume_id": volumeID,
		},
		"success": true,
	})
}

func (h *VolumeHandler) PruneVolumes(c *gin.Context) {
	var req struct {
		Filters map[string][]string `json:"filters"`
	}

	// Try to bind JSON body for filters (optional)
	c.ShouldBindJSON(&req)

	// Create filter args
	filterArgs := filters.NewArgs()
	if req.Filters != nil {
		for key, values := range req.Filters {
			for _, value := range values {
				filterArgs.Add(key, value)
			}
		}
	}

	var response volume.PruneReport
	var err error

	if len(filterArgs.Get("")) > 0 {
		// Use the method with custom filters if available
		response, err = h.dockerClient.PruneVolumesWithFilters(c.Request.Context(), filterArgs)
	} else {
		// Use the basic method
		response, err = h.dockerClient.PruneVolumes(c.Request.Context())
	}

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
			"message":         "Volumes pruned successfully",
			"volumes_deleted": response.VolumesDeleted,
			"space_reclaimed": response.SpaceReclaimed,
		},
		"success": true,
	})
}
