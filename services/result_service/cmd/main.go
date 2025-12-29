package main

import (
	"log"

	"github.com/DeadlyParkour777/code-checker/services/result_service/cmd/app"
	"github.com/DeadlyParkour777/code-checker/services/result_service/internal/config"
)

func main() {
	log.Println("Starting Result Service...")

	cfg := config.ConfigInit()

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application run failed: %v", err)
	}
}
