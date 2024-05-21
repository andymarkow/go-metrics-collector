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
	monitor        *monitor.Monitor
}

func NewAgent() (*Agent, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("newConfig: %w", err)
	}

	log, err := logger.NewZapLogger(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("logger.NewZapLogger: %w", err)
	}

	mon := monitor.NewMonitor(
		monitor.WithLogger(log),
		monitor.WithServerAddr(cfg.ServerAddr),
	)

	return &Agent{
		serverAddr:     cfg.ServerAddr,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
		log:            log,
		monitor:        mon,
	}, nil
}

func (a *Agent) Start() error {
	a.log.Sugar().Infof("Starting agent with server endpoint '%s'", a.serverAddr)
	a.log.Sugar().Infof("Polling interval: %s", a.pollInterval)
	a.log.Sugar().Infof("Reporting interval: %s", a.reportInterval)

	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)

	defer func() {
		pollTicker.Stop()
		reportTicker.Stop()
	}()

	for {
		select {
		case <-reportTicker.C:
			a.monitor.Push()
		case <-pollTicker.C:
			a.monitor.Collect()
		}
	}
}
