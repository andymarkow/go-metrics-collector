//nolint:gochecknoglobals
package main

import (
	"fmt"
	"log"

	"github.com/andymarkow/go-metrics-collector/internal/agent"
)

var (
	buildVersion = "N/A" // Build version number.
	buildDate    = "N/A" // Build creation date.
	buildCommit  = "N/A" // Build commit hash.
)

func main() {
	printBuildInfo()

	agnt, err := agent.NewAgent()
	if err != nil {
		log.Fatal(fmt.Errorf("agent.NewAgent: %w", err))
	}

	if err := agnt.Start(); err != nil {
		log.Fatal(fmt.Errorf("agent.Start: %w", err))
	}
}

// printBuildInfo prints the build version, date, and commit hash.
func printBuildInfo() {
	log.Println("Build version:", buildVersion)
	log.Println("Build date:", buildDate)
	log.Println("Build commit:", buildCommit)
}
