package main

import (
	"log"

	"github.com/DeadlyParkour777/code-checker/judge_service/cmd/app"
	"github.com/DeadlyParkour777/code-checker/judge_service/internal/config"
)

func main() {
	log.Println("Starting Judge Service...")

	cfg := config.ConfigInit()

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application run failed: %v", err)
	}
}
