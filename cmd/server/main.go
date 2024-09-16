//nolint:gochecknoglobals
package main

import (
	"fmt"
	"log"

	"github.com/andymarkow/go-metrics-collector/internal/server"
)

var (
	buildVersion = "N/A" // Build version number.
	buildDate    = "N/A" // Build creation date.
	buildCommit  = "N/A" // Build commit hash.
)

func main() {
	printBuildInfo()

	srv, err := server.NewServer()
	if err != nil {
		log.Fatal(fmt.Errorf("server.NewServer: %w", err))
	}

	if err := srv.Start(); err != nil {
		log.Fatal(fmt.Errorf("server.Start: %w", err))
	}
}

// printBuildInfo prints the build version, date, and commit hash.
func printBuildInfo() {
	log.Println("Build version:", buildVersion)
	log.Println("Build date:", buildDate)
	log.Println("Build commit:", buildCommit)
}
