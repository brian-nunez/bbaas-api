package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brian-nunez/bbaas-api/internal/httpserver"
)

func main() {
	port := getenvOrDefault("PORT", "8080")
	cdpManagerBaseURL := getenvOrDefault("CDP_MANAGER_BASE_URL", "http://127.0.0.1:8081")
	cdpPublicBaseURL := getenvOrDefault("CDP_PUBLIC_BASE_URL", "")
	dbDriver := getenvOrDefault("DB_DRIVER", "sqlite")
	dbDSN := getenvOrDefault("DB_DSN", "")

	server, err := httpserver.Bootstrap(httpserver.BootstrapConfig{
		StaticDirectories: map[string]string{
			"/assets": "./assets",
		},
		CDPManagerBaseURL: cdpManagerBaseURL,
		CDPPublicBaseURL:  cdpPublicBaseURL,
		DBDriver:          dbDriver,
		DBDSN:             dbDSN,
	})
	if err != nil {
		log.Fatalf("could not bootstrap server: %v", err)
	}

	go func() {
		err := server.Start(fmt.Sprintf(":%s", port))
		if err != nil && err.Error() != "http: Server closed" {
			log.Fatalf("could not start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Shutting down server...")
	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server exited cleanly")
}

func getenvOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
