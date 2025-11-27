package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Parse command-line flags
	flags := ParseFlags()

	// Load configuration
	config, err := LoadConfig(flags.ConfigPath, flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Print startup banner
	PrintStartupBanner(config)

	// Create and start server
	server := NewNTPServer(config)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nShutting down ChaosNTPd...")
		server.Stop()
		os.Exit(0)
	}()

	// Start server (blocking)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
