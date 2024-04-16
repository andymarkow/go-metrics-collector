package main

import (
	"fmt"
	"log"

	"github.com/andymarkow/go-metrics-collector/internal/server"
)

func main() {
	srv, err := server.NewServer()
	if err != nil {
		log.Fatal(fmt.Errorf("server.NewServer: %w", err))
	}

	if err := srv.Start(); err != nil {
		log.Fatal(fmt.Errorf("server.Start: %w", err))
	}
}
