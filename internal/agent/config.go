package agent

import (
	"flag"
	"strings"
)

type config struct {
	ServerAddr     string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
}

func newConfig() config {
	cfg := config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "server address")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval")
	flag.Parse()

	// Check if the URL does not start with "http://" or "https://"
	if !strings.HasPrefix(cfg.ServerAddr, "http://") &&
		!strings.HasPrefix(cfg.ServerAddr, "https://") {
		cfg.ServerAddr = "http://" + cfg.ServerAddr
	}

	// if err := env.Parse(&cfg); err != nil {
	// 	fmt.Printf("%+v\n", err)
	// }

	// fmt.Printf("serverAddr: %s\n", *serverAddr)
	// fmt.Printf("pollInterval: %d\n", *pollInterval)
	// fmt.Printf("reportInterval: %d\n", *reportInterval)

	// fmt.Printf("cfg: %+v\n", cfg)

	return cfg
}
