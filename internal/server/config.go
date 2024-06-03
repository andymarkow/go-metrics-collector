package server

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env"
)

type config struct {
	ServerAddr    string `env:"ADDRESS"`
	LogLevel      string `env:"LOG_LEVEL"`
	StoreFile     string `env:"FILE_STORAGE_PATH"`
	StoreInterval int    `env:"STORE_INTERVAL"`
	RestoreOnBoot bool   `env:"RESTORE"`
	DatabaseDSN   string `env:"DATABASE_DSN"`
	SignKey       string `env:"KEY"`
}

func newConfig() (config, error) {
	cfg := config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "server listening address [env:ADDRESS]")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log output level [env:LOG_LEVEL]")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string [env:DATABASE_DSN]")
	flag.StringVar(&cfg.SignKey, "k", "", "signing key [env:KEY]")
	flag.StringVar(&cfg.StoreFile, "f", "/tmp/metrics-db.json", "filepath to store metrics data to [env:FILE_STORAGE_PATH]")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "interval in seconds to store metrics data into file [env:STORE_INTERVAL]")
	flag.BoolVar(&cfg.RestoreOnBoot, "r", true, "whether or not to restore metrics data from file [env:RESTORE]")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("env.Parse: %w", err)
	}

	return cfg, nil
}
