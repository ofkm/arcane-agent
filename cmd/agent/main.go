package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/ofkm/arcane-agent/internal/agent"
	"github.com/ofkm/arcane-agent/internal/config"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and start agent
	agent := agent.New(cfg)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("Received shutdown signal")
		agent.Stop()
	}()

	// Start agent (blocks until shutdown)
	if err := agent.Start(); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	log.Printf("Agent stopped")
}
