package main

import (
	"embed"
	"log"
	"net"
	"os"
	"pa11y-go-wrapper/internal/analysis"
	"pa11y-go-wrapper/internal/api"
	"pa11y-go-wrapper/internal/discovery"
)

//go:embed frontend
var frontendAssets embed.FS

func main() {
	// Initialize the analysis service
	analysisService := analysis.NewService(100) // Queue size of 100
	discoveryService := discovery.NewService()

	// Start the background worker
	worker := analysis.NewWorker(analysisService)
	worker.Start()

	// Create and run the Gin server
	router := api.NewRouter(analysisService, discoveryService, frontendAssets)

	addr := getServerAddr()

	log.Printf("Starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func getServerAddr() string {
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	port := os.Getenv("PORT")
	if port != "" {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			// addr is just a port, so we can ignore the error and use an empty host
			host = ""
		}
		addr = net.JoinHostPort(host, port)
	}
	return addr
}
