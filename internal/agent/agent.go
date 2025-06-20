package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/docker"
	"github.com/ofkm/arcane-agent/internal/tasks"
)

type Agent struct {
	config       *config.Config
	httpClient   *HTTPClient
	dockerClient *docker.Client
	taskManager  *tasks.Manager

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	shutdown  chan struct{}
	startTime time.Time
}

func New(cfg *config.Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	dockerClient := docker.NewClient()
	taskManager := tasks.NewManager(dockerClient, cfg)
	httpClient := NewHTTPClient(cfg, taskManager)

	return &Agent{
		config:       cfg,
		httpClient:   httpClient,
		dockerClient: dockerClient,
		taskManager:  taskManager,
		ctx:          ctx,
		cancel:       cancel,
		shutdown:     make(chan struct{}),
		startTime:    time.Now(),
	}
}

func (a *Agent) Start() error {
	log.Printf("Starting Arcane Agent %s", a.config.AgentID)

	// Start HTTP client (handles registration, heartbeat, and task polling)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.httpClient.Start(a.ctx); err != nil {
			log.Printf("HTTP client error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-a.shutdown

	log.Printf("Shutting down agent...")
	a.cancel()
	a.wg.Wait()

	return nil
}

func (a *Agent) Stop() {
	select {
	case <-a.shutdown:
		// Already closed
		return
	default:
		close(a.shutdown)
	}
}
