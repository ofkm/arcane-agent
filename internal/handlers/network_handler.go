package handlers

import (
	"net/http"

	"github.com/docker/docker/api/types/network"
	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
)

type NetworkHandler struct {
	dockerClient *docker.Client
}

func NewNetworkHandler(dockerClient *docker.Client) *NetworkHandler {
	return &NetworkHandler{
		dockerClient: dockerClient,
	}
}

func (h *NetworkHandler) ListNetworks(c *gin.Context) {
	networks, err := h.dockerClient.ListNetworks(c.Request.Context())
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
			"networks": networks,
			"total":    len(networks),
		},
		"success": true,
	})
}

func (h *NetworkHandler) GetNetwork(c *gin.Context) {
	networkID := c.Param("id")
	network, err := h.dockerClient.GetNetwork(c.Request.Context(), networkID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    network,
		"success": true,
	})
}

func (h *NetworkHandler) CreateNetwork(c *gin.Context) {
	var req struct {
		Name       string            `json:"name" binding:"required"`
		Driver     string            `json:"driver"`
		Internal   bool              `json:"internal"`
		Attachable bool              `json:"attachable"`
		Ingress    bool              `json:"ingress"`
		EnableIPv6 bool              `json:"enableIPv6"`
		Options    map[string]string `json:"options"`
		Labels     map[string]string `json:"labels"`
		IPAM       *network.IPAM     `json:"ipam"`
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
		req.Driver = "bridge"
	}

	options := network.CreateOptions{
		Driver:     req.Driver,
		Options:    req.Options,
		Labels:     req.Labels,
		Internal:   req.Internal,
		Attachable: req.Attachable,
		Ingress:    req.Ingress,
		EnableIPv6: &req.EnableIPv6,
		IPAM:       req.IPAM,
	}

	response, err := h.dockerClient.CreateNetwork(c.Request.Context(), req.Name, options)
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
			"message":    "Network created successfully",
			"network_id": response.ID,
			"name":       req.Name,
			"warning":    response.Warning,
		},
		"success": true,
	})
}

func (h *NetworkHandler) DeleteNetwork(c *gin.Context) {
	networkID := c.Param("id")

	err := h.dockerClient.RemoveNetwork(c.Request.Context(), networkID)
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
			"message":    "Network deleted successfully",
			"network_id": networkID,
		},
		"success": true,
	})
}

func (h *NetworkHandler) ConnectContainer(c *gin.Context) {
	networkID := c.Param("id")

	var req struct {
		Container      string                    `json:"container" binding:"required"`
		EndpointConfig *network.EndpointSettings `json:"endpointConfig"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	err := h.dockerClient.ConnectContainerToNetwork(c.Request.Context(), networkID, req.Container, req.EndpointConfig)
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
			"message":      "Container connected to network successfully",
			"network_id":   networkID,
			"container_id": req.Container,
		},
		"success": true,
	})
}

func (h *NetworkHandler) DisconnectContainer(c *gin.Context) {
	networkID := c.Param("id")

	var req struct {
		Container string `json:"container" binding:"required"`
		Force     bool   `json:"force"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":    nil,
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	err := h.dockerClient.DisconnectContainerFromNetwork(c.Request.Context(), networkID, req.Container, req.Force)
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
			"message":      "Container disconnected from network successfully",
			"network_id":   networkID,
			"container_id": req.Container,
		},
		"success": true,
	})
}

func (h *NetworkHandler) PruneNetworks(c *gin.Context) {
	response, err := h.dockerClient.PruneNetworks(c.Request.Context())
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
			"message":          "Networks pruned successfully",
			"networks_deleted": response.NetworksDeleted,
		},
		"success": true,
	})
}
