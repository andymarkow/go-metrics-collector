package agent

import (
	"time"

	"github.com/andymarkow/go-metrics-collector/internal/monitor"
)

type Agent struct {
	serverAddr     string
	pollInterval   time.Duration
	reportInterval time.Duration
}

func NewAgent() *Agent {
	cfg := newConfig()

	return &Agent{
		serverAddr:     cfg.ServerAddr,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
	}
}

func (a *Agent) Start() error {
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
