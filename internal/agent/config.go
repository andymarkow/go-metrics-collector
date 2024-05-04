package agent

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env"
)

type config struct {
	ServerAddr     string `env:"ADDRESS"`
	LogLevel       string `env:"LOG_LEVEL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
}

func newConfig() (config, error) {
	cfg := config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "server endpoint address [env:ADDRESS]")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log output level [env:LOG_LEVEL]")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval in seconds [env:POLL_INTERVAL]")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval in seconds [env:REPORT_INTERVAL]")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("env.Parse: %w", err)
	}

	// Check if the URL does not start with "http://" or "https://"
	if !strings.HasPrefix(cfg.ServerAddr, "http://") &&
		!strings.HasPrefix(cfg.ServerAddr, "https://") {
		cfg.ServerAddr = "http://" + cfg.ServerAddr
	}

	return cfg, nil
}
