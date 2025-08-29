package main

import (
	"log"

	"github.com/DeadlyParkour777/code-checker/submission_service/cmd/app"
	"github.com/DeadlyParkour777/code-checker/submission_service/internal/config"
)

func main() {
	log.Println("Starting Submission Service...")

	cfg := config.ConfigInit()

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
