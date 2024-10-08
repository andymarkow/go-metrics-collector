package agent

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env"
)

// config represents the agent configuration.
//
//nolint:tagalign,tagliatelle
type config struct {
	ConfigFile     string `env:"CONFIG" json:"config"`
	ServerAddr     string `env:"ADDRESS" json:"address"`
	LogLevel       string `env:"LOG_LEVEL" json:"log_level"`
	SignKey        string `env:"KEY" json:"key"`
	CryptoKey      string `env:"CRYPTO_KEY" json:"crypto_key"`
	PollInterval   int    `env:"POLL_INTERVAL" json:"poll_interval"`
	ReportInterval int    `env:"REPORT_INTERVAL" json:"report_interval"`
	RateLimit      int    `env:"RATE_LIMIT" json:"rate_limit"`
	UseGrpc        bool   `env:"USE_GRPC" json:"use_grpc"`
}

// newConfig creates a new config for agent.
func newConfig() (config, error) {
	cfg := config{}

	flag.StringVar(&cfg.ConfigFile, "c", "./config/agent.json", "path to config file [env:CONFIG]")
	flag.StringVar(&cfg.ServerAddr, "a", "", "server endpoint address [env:ADDRESS]")
	flag.StringVar(&cfg.LogLevel, "lv", "", "log output level [env:LOG_LEVEL]")
	flag.StringVar(&cfg.SignKey, "k", "", "signing key [env:KEY]")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to RSA public key file to encrypt messages to Server [env:CRYPTO_KEY]")
	flag.IntVar(&cfg.PollInterval, "p", 0, "poll interval in seconds [env:POLL_INTERVAL]")
	flag.IntVar(&cfg.ReportInterval, "r", 0, "report interval in seconds [env:REPORT_INTERVAL]")
	flag.IntVar(&cfg.RateLimit, "l", 0, "the number of simultaneous outgoing requests to the server [env:RATE_LIMIT]")
	flag.BoolVar(&cfg.UseGrpc, "grpc", false, "whether or not to use gRPC [env:USE_GRPC]")
	flag.Parse()

	// Highest precedence for environment variables.
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("env.Parse: %w", err)
	}

	// Lowest precedence for configuration file.
	if err := readConfigFile(cfg.ConfigFile, &cfg); err != nil {
		return cfg, fmt.Errorf("readConfigFile: %w", err)
	}

	return cfg, nil
}

func readConfigFile(file string, cfg *config) error {
	f, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("os.ReadFile: %w", err)
	}

	fileCfg := new(config)

	if err := json.Unmarshal(f, fileCfg); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	if cfg.CryptoKey == "" {
		if fileCfg.CryptoKey == "" {
			cfg.CryptoKey = fileCfg.CryptoKey
		}
	}

	if cfg.LogLevel == "" {
		if fileCfg.LogLevel == "" {
			cfg.LogLevel = "info"
		} else {
			cfg.LogLevel = fileCfg.LogLevel
		}
	}

	if cfg.PollInterval == 0 {
		if fileCfg.PollInterval == 0 {
			cfg.PollInterval = 2
		} else {
			cfg.PollInterval = fileCfg.PollInterval
		}
	}

	if cfg.ReportInterval == 0 {
		if fileCfg.ReportInterval == 0 {
			cfg.ReportInterval = 10
		} else {
			cfg.ReportInterval = fileCfg.ReportInterval
		}
	}

	if cfg.RateLimit == 0 {
		if fileCfg.RateLimit == 0 {
			cfg.RateLimit = 1
		} else {
			cfg.RateLimit = fileCfg.RateLimit
		}
	}

	if cfg.ServerAddr == "" {
		if fileCfg.ServerAddr == "" {
			cfg.ServerAddr = "localhost:8080"
		} else {
			cfg.ServerAddr = fileCfg.ServerAddr
		}
	}

	if cfg.SignKey == "" {
		cfg.SignKey = fileCfg.SignKey
	}

	if !cfg.UseGrpc {
		if fileCfg.UseGrpc {
			cfg.UseGrpc = fileCfg.UseGrpc
		}
	}

	return nil
}
