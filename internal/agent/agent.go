package agent

import (
	"fmt"
	"time"

	"github.com/andymarkow/go-metrics-collector/internal/logger"
	"github.com/andymarkow/go-metrics-collector/internal/monitor"
	"go.uber.org/zap"
)

type Agent struct {
	serverAddr     string
	pollInterval   time.Duration
	reportInterval time.Duration
	log            *zap.Logger
}

func NewAgent() (*Agent, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("newConfig: %w", err)
	}

	log, err := logger.NewZapLogger(&logger.Config{
		Level: cfg.LogLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("logger.NewZapLogger: %w", err)
	}

	return &Agent{
		serverAddr:     cfg.ServerAddr,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
		log:            log,
	}, nil
}

func (a *Agent) Start() error {
	a.log.Sugar().Infof("Starting agent with server endpoint '%s'", a.serverAddr)
	a.log.Sugar().Infof("Polling interval: %s", a.pollInterval)
	a.log.Sugar().Infof("Reporting interval: %s", a.reportInterval)

	mon := monitor.NewMonitor(&monitor.Config{
		ServerAddr: a.serverAddr,
		Logger:     a.log,
	})

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
