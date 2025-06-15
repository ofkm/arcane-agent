package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ofkm/arcane-agent/internal/config"
	"github.com/ofkm/arcane-agent/internal/docker"
	"github.com/ofkm/arcane-agent/internal/tasks"
)

type Agent struct {
	config       *config.Config
	wsClient     *WebSocketClient
	httpClient   *HTTPClient // Keep as fallback
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

	var wsClient *WebSocketClient
	if cfg.UseWebSocket {
		wsClient = NewWebSocketClient(cfg, taskManager)
	}

	httpClient := NewHTTPClient(cfg, taskManager) // Always create for fallback/registration

	return &Agent{
		config:       cfg,
		wsClient:     wsClient,
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

	if a.config.UseWebSocket && a.wsClient != nil {
		log.Printf("Using WebSocket communication")

		// Register agent via HTTP first (required for WebSocket auth)
		if err := a.httpClient.registerAgent(); err != nil {
			return fmt.Errorf("failed to register agent: %w", err)
		}
		log.Printf("Agent registered successfully")

		// Start WebSocket client
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			if err := a.wsClient.Start(a.ctx); err != nil {
				log.Printf("WebSocket client error: %v", err)
				log.Printf("Falling back to HTTP polling...")

				// Fallback to HTTP if WebSocket fails
				if err := a.httpClient.startPolling(a.ctx); err != nil {
					log.Printf("HTTP polling error: %v", err)
				}
			}
		}()
	} else {
		log.Printf("Using HTTP polling communication")

		// Use HTTP polling only
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			if err := a.httpClient.Start(a.ctx); err != nil {
				log.Printf("HTTP client error: %v", err)
			}
		}()
	}

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
