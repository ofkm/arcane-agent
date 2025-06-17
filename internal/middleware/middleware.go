package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
)

func APIKeyMiddleware(expectedAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != expectedAPIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"data":    nil,
				"success": false,
				"error":   "Unauthorized",
			})
			return
		}
		c.Next()
	}
}

func DockerAvailabilityMiddleware(dockerClient *docker.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if dockerClient == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"data":    nil,
				"success": false,
				"error":   "Docker not available",
			})
			return
		}
		c.Next()
	}
}
