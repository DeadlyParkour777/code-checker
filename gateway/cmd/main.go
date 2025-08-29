package main

import (
	"log"

	"github.com/DeadlyParkour777/code-checker/gateway/cmd/app"
	"github.com/DeadlyParkour777/code-checker/gateway/internal/config"
)

// @title Code Checker API Gateway
// @version 1.0

// @host localhost:8000
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
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
