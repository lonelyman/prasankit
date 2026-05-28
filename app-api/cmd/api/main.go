package main

import (
	"log"

	"prasankit-api/internal/bootstrap"
	"prasankit-api/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	app, err := bootstrap.InitializeApp(cfg)
	if err != nil {
		log.Fatalf("app initialization error: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
