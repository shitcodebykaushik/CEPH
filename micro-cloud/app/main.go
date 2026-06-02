package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/shiiit/micro-cloud/internal/api"
	"github.com/shiiit/micro-cloud/internal/database"
	"github.com/shiiit/micro-cloud/internal/orchestrator"
	"github.com/shiiit/micro-cloud/internal/storage"
)

func main() {
	fmt.Println("=== Micro-Cloud Orchestrator Starting ===")

	dbPath := os.Getenv("MICRO_CLOUD_DB")
	if dbPath == "" {
		dbPath = "/app/data/micro-cloud.db"
	}

	if err := database.InitDB(dbPath); err != nil {
		log.Fatalf("[DB] Init failed: %v", err)
	}
	defer database.CloseDB()

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "127.0.0.1:9000"
	}
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	if minioAccessKey == "" {
		minioAccessKey = "minioadmin"
	}
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	if minioSecretKey == "" {
		minioSecretKey = "minioadmin"
	}

	minioMgr, err := storage.NewMinioManager(minioEndpoint, minioAccessKey, minioSecretKey, false)
	if err != nil {
		log.Printf("[MINIO] Connection failed (non-fatal): %v", err)
		minioMgr = nil
	}

	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("[DOCKER] Client creation failed: %v", err)
	}
	defer dockerClient.Close()

	version, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		log.Printf("[DOCKER] Could not get version (non-fatal): %v", err)
	} else {
		fmt.Printf("[DOCKER] ✓ Docker version: %s (API: %s)\n", version.Version, version.APIVersion)
	}

	eng := orchestrator.NewEngine(minioMgr, dockerClient)
	handler := api.NewHandler(eng)
	router := api.SetupRouter(handler)

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		fmt.Printf("[API] Listening on :%s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[API] Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n=== Shutting down ===")
	server.Close()
}
