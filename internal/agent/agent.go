package agent

import (
	"fmt"
	"log"
	"time"

	"github.com/andymarkow/go-metrics-collector/internal/monitor"
)

type Agent struct {
	serverAddr     string
	pollInterval   time.Duration
	reportInterval time.Duration
}

func NewAgent() (*Agent, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("newConfig: %w", err)
	}

	return &Agent{
		serverAddr:     cfg.ServerAddr,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
	}, nil
}

func (a *Agent) Start() error {
	log.Printf("Starting agent with server endpoint %q\n", a.serverAddr)
	log.Printf("Polling interval: %s\n", a.pollInterval)
	log.Printf("Reporting interval: %s\n", a.reportInterval)

	mon := monitor.NewMonitor(a.serverAddr)

	pollTicket := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)

	defer func() {
		pollTicket.Stop()
		reportTicker.Stop()
	}()

	for {
		select {
		case <-reportTicker.C:
			mon.Push()
		case <-pollTicket.C:
			mon.Collect()
		}
	}
}
