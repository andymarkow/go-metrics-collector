package server

import (
	"flag"
)

type config struct {
	ServerAddr string `env:"ADDRESS"`
}

func newConfig() config {
	cfg := config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "server listening address [env:ADDRESS]")
	flag.Parse()

	return cfg
}
