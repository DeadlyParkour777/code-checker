package main

import (
	"log"

	"github.com/DeadlyParkour777/code-checker/services/gateway/cmd/app"
	"github.com/DeadlyParkour777/code-checker/services/gateway/internal/config"
)

func main() {
	cfg := config.ConfigInit()
	log.Println("Configuration loaded")

	server, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}
	log.Println("API Server created")

	if err := server.Run(); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}
