package agent

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ofkm/arcane-agent/internal/api"
	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/docker"
)

type Agent struct {
	config       *config.Config
	dockerClient *docker.Client
	apiServer    *http.Server

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	shutdown  chan struct{}
	startTime time.Time

	status string
	mu     sync.RWMutex
}

func New(cfg *config.Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Printf("Warning: Docker client creation failed: %v", err)
		dockerClient = nil
	}

	return &Agent{
		config:       cfg,
		dockerClient: dockerClient,
		ctx:          ctx,
		cancel:       cancel,
		shutdown:     make(chan struct{}),
		startTime:    time.Now(),
		status:       "initializing",
	}
}

func (a *Agent) Start() error {
	a.setStatus("starting")
	log.Printf("Starting Arcane Agent %s (version: %s)", a.config.AgentID, a.config.Version)

	// Validate Docker
	if a.dockerClient == nil || !a.dockerClient.IsDockerAvailable() {
		log.Printf("Warning: Docker is not available")
	} else {
		log.Printf("Docker connection successful")
	}

	// Setup API server
	router := api.NewRouter(a.config, a.dockerClient)
	listenAddr := fmt.Sprintf("%s:%d", a.config.AgentListenAddress, a.config.AgentPort)

	a.apiServer = &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	// Start API server
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		log.Printf("Agent API server listening on %s", listenAddr)
		a.setStatus("running")

		if err := a.apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Agent API server error: %v", err)
		}
		log.Println("Agent API server shut down.")
	}()

	log.Printf("Agent started successfully")

	// Wait for shutdown
	<-a.shutdown

	log.Printf("Shutting down agent...")
	a.setStatus("stopping")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Agent API server shutdown error: %v", err)
	}

	a.cancel()
	a.wg.Wait()
	a.setStatus("stopped")

	if a.dockerClient != nil {
		a.dockerClient.Close()
	}

	log.Println("Agent stopped gracefully.")
	return nil
}

func (a *Agent) Stop() {
	log.Println("Stop called on agent.")
	select {
	case <-a.shutdown:
		return
	default:
		close(a.shutdown)
	}
}

func (a *Agent) GetStatus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *Agent) setStatus(status string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
	log.Printf("Agent status: %s", status)
}
