package main

import (
	"fmt"
	"log"

	"github.com/andymarkow/go-metrics-collector/internal/agent"
)

func main() {
	agnt, err := agent.NewAgent()
	if err != nil {
		log.Fatal(fmt.Errorf("agent.NewAgent: %w", err))
	}

	if err := agnt.Start(); err != nil {
		log.Fatal(fmt.Errorf("agent.Start: %w", err))
	}
}
