package agent

import (
	"time"

	"github.com/andymarkow/go-metrics-collector/internal/monitor"
)

type Agent struct {
	pollInterval   time.Duration
	reportInterval time.Duration
}

func NewAgent() *Agent {
	return &Agent{
		pollInterval:   2 * time.Second,
		reportInterval: 10 * time.Second,
	}
}

func (a *Agent) Start() error {
	mon := monitor.NewMonitor()

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
