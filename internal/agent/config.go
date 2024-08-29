package agent

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env"
)

// config represents the agent configuration.
type config struct {
	ServerAddr     string `env:"ADDRESS"`
	LogLevel       string `env:"LOG_LEVEL"`
	SignKey        string `env:"KEY"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

// newConfig creates a new config for agent.
func newConfig() (config, error) {
	cfg := config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "server endpoint address [env:ADDRESS]")
	flag.StringVar(&cfg.LogLevel, "lv", "info", "log output level [env:LOG_LEVEL]")
	flag.StringVar(&cfg.SignKey, "k", "", "signing key [env:KEY]")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval in seconds [env:POLL_INTERVAL]")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval in seconds [env:REPORT_INTERVAL]")
	flag.IntVar(&cfg.RateLimit, "l", 1, "the number of simultaneous outgoing requests to the server [env:RATE_LIMIT]")
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
