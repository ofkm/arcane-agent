package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/config"
)

type StatusHandler struct {
	config *config.Config
}

func NewStatusHandler(cfg *config.Config) *StatusHandler {
	return &StatusHandler{
		config: cfg,
	}
}

func (h *StatusHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "running",
		"agent_id": h.config.AgentID,
		"version":  h.config.Version,
	})
}
