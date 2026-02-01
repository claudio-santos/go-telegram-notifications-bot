package main

import (
	"fmt"
	"log"
	"net/http"

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

	// Initialize handlers
	handlers := internal.NewHandlers(configManager)

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

	fmt.Printf("Server starting on %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
