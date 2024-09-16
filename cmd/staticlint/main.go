// Package main provides the entry point for the staticlint command.
package main

import "github.com/andymarkow/go-metrics-collector/internal/staticlint"

func main() {
	staticLint := staticlint.NewStaticlint()
	staticLint.Run()
}
