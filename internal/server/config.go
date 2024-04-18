package server

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env"
)

type config struct {
	ServerAddr string `env:"ADDRESS"`
	LogLevel   string `env:"LOG_LEVEL"`
}

func newConfig() (config, error) {
	cfg := config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "server listening address [env:ADDRESS]")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log output level [env:LOG_LEVEL]")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("env.Parse: %w", err)
	}

	return cfg, nil
}
