package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env"
)

// config represents the server configuration.
//
//nolint:tagalign,tagliatelle
type config struct {
	ConfigFile     string `env:"CONFIG" json:"config"`
	ServerAddr     string `env:"ADDRESS" json:"address"`
	GrpcServerAddr string `env:"GRPC_ADDRESS" json:"grpc_address"`
	LogLevel       string `env:"LOG_LEVEL" json:"log_level"`
	DatabaseDSN    string `env:"DATABASE_DSN" json:"database_dsn"`
	SignKey        string `env:"KEY" json:"sign_key"`
	CryptoKey      string `env:"CRYPTO_KEY" json:"crypto_key"`
	TrustedSubnet  string `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
	StoreFile      string `env:"FILE_STORAGE_PATH" json:"store_file"`
	StoreInterval  int    `env:"STORE_INTERVAL" json:"store_interval"`
	RestoreOnBoot  bool   `env:"RESTORE" json:"restore"`
}

// newConfig creates a new config for the server.
//
// It uses both environment variables and command line flags to populate the
// config struct. If any of the environment variables or command line flags are
// not set, it will use default values.
//
// If there is an error while parsing the environment variables, it will return
// an error.
func newConfig() (config, error) {
	cfg := config{}

	flag.StringVar(&cfg.ConfigFile, "c", "./config/server.json", "path to config file [env:CONFIG]")
	flag.StringVar(&cfg.ServerAddr, "a", "", "server listening address [env:ADDRESS]")
	flag.StringVar(&cfg.GrpcServerAddr, "g", "", "gRPC server listening address [env:GRPC_ADDRESS]")
	flag.StringVar(&cfg.LogLevel, "l", "", "log output level [env:LOG_LEVEL]")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string [env:DATABASE_DSN]")
	flag.StringVar(&cfg.SignKey, "k", "", "signing key [env:KEY]")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to RSA private key file to decrypt messages from Agent [env:CRYPTO_KEY]")
	flag.StringVar(&cfg.TrustedSubnet, "t", "", "trusted subnet [env:TRUSTED_SUBNET]")
	flag.StringVar(&cfg.StoreFile, "f", "", "filepath to store metrics data to [env:FILE_STORAGE_PATH]")
	flag.IntVar(&cfg.StoreInterval, "i", 0, "interval in seconds to store metrics data into file [env:STORE_INTERVAL]")
	flag.BoolVar(&cfg.RestoreOnBoot, "r", false, "whether or not to restore metrics data from file [env:RESTORE]")
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

	if cfg.DatabaseDSN == "" {
		cfg.DatabaseDSN = fileCfg.DatabaseDSN
	}

	if cfg.LogLevel == "" {
		if fileCfg.LogLevel == "" {
			cfg.LogLevel = "info"
		} else {
			cfg.LogLevel = fileCfg.LogLevel
		}
	}

	if cfg.ServerAddr == "" {
		if fileCfg.ServerAddr == "" {
			cfg.ServerAddr = "localhost:8080"
		} else {
			cfg.ServerAddr = fileCfg.ServerAddr
		}
	}

	if cfg.GrpcServerAddr == "" {
		if fileCfg.GrpcServerAddr == "" {
			cfg.GrpcServerAddr = "localhost:50051"
		} else {
			cfg.GrpcServerAddr = fileCfg.GrpcServerAddr
		}
	}

	if cfg.SignKey == "" {
		cfg.SignKey = fileCfg.SignKey
	}

	if cfg.TrustedSubnet == "" {
		cfg.TrustedSubnet = fileCfg.TrustedSubnet
	}

	if cfg.StoreFile == "" {
		if fileCfg.StoreFile == "" {
			cfg.StoreFile = "/tmp/metrics-db.json"
		} else {
			cfg.StoreFile = fileCfg.StoreFile
		}
	}

	if cfg.StoreInterval == 0 {
		if fileCfg.StoreInterval == 0 {
			cfg.StoreInterval = 300
		} else {
			cfg.StoreInterval = fileCfg.StoreInterval
		}
	}

	if !cfg.RestoreOnBoot {
		if fileCfg.RestoreOnBoot {
			cfg.RestoreOnBoot = true
		} else {
			cfg.RestoreOnBoot = fileCfg.RestoreOnBoot
		}
	}

	return nil
}
