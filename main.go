package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-telegram-notifications-bot/internal"
)

func main() {
	// Initialize config manager
	configManager := internal.NewConfigManager()

	// Load configuration
	err := configManager.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	dbManager, err := internal.NewDBManager(configManager.Config.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbManager.Close()

	// Initialize scheduler
	scheduler := internal.NewFeedScheduler(configManager, dbManager)

	// Start the scheduler
	scheduler.Start()

	// Start the cleanup routine
	scheduler.StartCleanupRoutine()

	// Initialize handlers
	handlers := internal.NewHandlers(configManager, scheduler)

	// Setup router
	r := internal.Router(handlers)

	// Extract port from server config (format: ":8080")
	port := ":8080" // default port
	if configManager.Config.Server != "" {
		port = configManager.Config.Server
		if port[0] != ':' {
			port = ":" + port
		}
	}

	// Create a channel to listen for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	fmt.Printf("Server starting on %s\n", port)

	// Start server in a goroutine
	go func() {
		log.Fatal(http.ListenAndServe(port, r))
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down gracefully...")

	// Stop the scheduler
	scheduler.Stop()

	log.Println("Server stopped")
}
