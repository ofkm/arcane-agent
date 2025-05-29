package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	agent := agent.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan // First signal wait
		fmt.Println("\nShutting down agent...")
		cancel()
	}()

	if err := agent.Start(ctx); err != nil {
		log.Fatalf("Agent failed to start: %v", err)
	}

	// Give some time for graceful shutdown
	shutdownTimeout := 10 * time.Second
	shutdownTimer := time.NewTimer(shutdownTimeout)
	defer shutdownTimer.Stop()

	select {
	case <-shutdownTimer.C:
		log.Println("Shutdown timeout exceeded, forcing exit")
		os.Exit(1)
	case <-ctx.Done():
		log.Println("Agent shutdown complete")
	}
}
