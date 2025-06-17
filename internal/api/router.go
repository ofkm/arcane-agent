package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/docker"
	"github.com/ofkm/arcane-agent/internal/handlers"
	"github.com/ofkm/arcane-agent/internal/middleware"
)

func NewRouter(cfg *config.Config, dockerClient *docker.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	if cfg.APIKey != "" {
		router.Use(middleware.APIKeyMiddleware(cfg.APIKey))
	}

	// Initialize handlers
	statusHandler := handlers.NewStatusHandler(cfg)
	containerHandler := handlers.NewContainerHandler(dockerClient)
	dockerHandler := handlers.NewDockerHandler(dockerClient)
	imageHandler := handlers.NewImageHandler(dockerClient)

	api := router.Group("/api")
	{
		setupStatusRoutes(api, statusHandler)
		setupContainerRoutes(api, containerHandler, dockerClient)
		setupDockerRoutes(api, dockerHandler, dockerClient)
		setupImageRoutes(api, imageHandler, dockerClient)
		setupNetworkRoutes(api, handlers.NewNetworkHandler(dockerClient))
		setupVolumeRoutes(api, handlers.NewVolumeHandler(dockerClient), dockerClient)
	}

	return router
}

// Status routes
func setupStatusRoutes(api *gin.RouterGroup, statusHandler *handlers.StatusHandler) {
	api.GET("/status", statusHandler.GetStatus)
}

// Container routes
func setupContainerRoutes(api *gin.RouterGroup, containerHandler *handlers.ContainerHandler, dockerClient *docker.Client) {
	containers := api.Group("/containers")
	containers.Use(middleware.DockerAvailabilityMiddleware(dockerClient))
	{
		containers.GET("", containerHandler.ListContainers)
		containers.GET("/:id", containerHandler.GetContainer)
		containers.POST("/:id/start", containerHandler.StartContainer)
		containers.POST("/:id/stop", containerHandler.StopContainer)
		containers.POST("/:id/restart", containerHandler.RestartContainer)
	}
}

// Docker system routes
func setupDockerRoutes(api *gin.RouterGroup, dockerHandler *handlers.DockerHandler, dockerClient *docker.Client) {
	docker := api.Group("/docker")
	docker.Use(middleware.DockerAvailabilityMiddleware(dockerClient))
	{
		docker.GET("/info", dockerHandler.GetDockerInfo)
	}
}

// Image routes
func setupImageRoutes(api *gin.RouterGroup, imageHandler *handlers.ImageHandler, dockerClient *docker.Client) {
	images := api.Group("/images")
	images.Use(middleware.DockerAvailabilityMiddleware(dockerClient))
	{
		images.GET("", imageHandler.ListImages)
		images.POST("", imageHandler.CreateImage)
		images.POST("/pull", imageHandler.Pull)
		images.GET("/:id", imageHandler.GetImage)
		images.DELETE("/:id", imageHandler.DeleteImage)
		images.POST("/:id/tag", imageHandler.TagImage)
		images.POST("/:id/push", imageHandler.PushImage)
	}
}

func setupNetworkRoutes(router *gin.RouterGroup, networkHandler *handlers.NetworkHandler) {
	networks := router.Group("/networks")
	{
		networks.GET("", networkHandler.ListNetworks)
		networks.POST("", networkHandler.CreateNetwork)
		networks.GET("/:id", networkHandler.GetNetwork)
		networks.DELETE("/:id", networkHandler.DeleteNetwork)
		networks.POST("/:id/connect", networkHandler.ConnectContainer)
		networks.POST("/:id/disconnect", networkHandler.DisconnectContainer)
		networks.POST("/prune", networkHandler.PruneNetworks)
	}
}

func setupVolumeRoutes(api *gin.RouterGroup, volumeHandler *handlers.VolumeHandler, dockerClient *docker.Client) {
	volumes := api.Group("/volumes")
	volumes.Use(middleware.DockerAvailabilityMiddleware(dockerClient))
	{
		volumes.GET("", volumeHandler.ListVolumes)
		volumes.POST("", volumeHandler.CreateVolume)
		volumes.GET("/:id", volumeHandler.GetVolume)
		volumes.DELETE("/:id", volumeHandler.DeleteVolume)
		volumes.POST("/prune", volumeHandler.PruneVolumes)
	}
}
