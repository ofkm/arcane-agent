package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/docker"
)

// APIKeyMiddleware for API key authentication
func APIKeyMiddleware(expectedAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != expectedAPIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	}
}

// DockerAvailabilityMiddleware checks if Docker client is available
func DockerAvailabilityMiddleware(dockerClient *docker.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if dockerClient == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Docker not available"})
			return
		}
		c.Next()
	}
}
