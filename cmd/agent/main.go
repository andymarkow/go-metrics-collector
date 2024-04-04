package main

import (
	"github.com/andymarkow/go-metrics-collector/internal/agent"
)

func main() {
	app := agent.NewAgent()
	if err := app.Start(); err != nil {
		panic(err)
	}
}
