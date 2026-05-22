package main

import (
	"log"

	"mock-server-go/internal/server"
)

func main() {
	cfg := server.ConfigFromEnv()
	if err := server.Run(cfg); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
