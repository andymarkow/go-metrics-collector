package main

import "github.com/andymarkow/go-metrics-collector/internal/server"

func main() {
	srv := server.NewServer()
	if err := srv.Start(); err != nil {
		panic(err)
	}
}
