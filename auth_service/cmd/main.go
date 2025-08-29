package main

import (
	"log"

	"github.com/DeadlyParkour777/code-checker/auth_service/cmd/app"
	"github.com/DeadlyParkour777/code-checker/auth_service/internal/config"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.ConfigInit()

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to init app: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Failed to run app: %v", err)
	}
}
