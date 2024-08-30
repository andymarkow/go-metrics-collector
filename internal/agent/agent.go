// Package agent provides a metrics collector and reporter agent.
package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/logger"
	"github.com/andymarkow/go-metrics-collector/internal/monitor"
)

// Agent represents a metrics agent that collects and reports metrics.
type Agent struct {
	serverAddr     string           // ServerAddr is the address of the server.
	pollInterval   time.Duration    // PollInterval is the interval at which metrics are collected.
	reportInterval time.Duration    // ReportInterval is the interval at which metrics are reported.
	log            *zap.Logger      // Log is the logger instance used for logging.
	monitor        *monitor.Monitor // Monitor is the monitor instance used for monitoring.
}

// NewAgent creates a new agent instance.
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
		monitor.WithSignKey([]byte(cfg.SignKey)),
		monitor.WithPollInterval(time.Duration(cfg.PollInterval)*time.Second),
		monitor.WithReportInterval(time.Duration(cfg.ReportInterval)*time.Second),
		monitor.WithRateLimit(cfg.RateLimit),
	)

	return &Agent{
		serverAddr:     cfg.ServerAddr,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
		log:            log,
		monitor:        mon,
	}, nil
}

// Start starts the agent intance.
func (a *Agent) Start() error {
	a.log.Sugar().Infof("Starting agent with server endpoint '%s'", a.serverAddr)
	a.log.Sugar().Infof("Polling interval: %s", a.pollInterval)
	a.log.Sugar().Infof("Reporting interval: %s", a.reportInterval)

	wg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		a.monitor.RunCollector(ctx)
	}(wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		a.monitor.RunCollectorGopsutils(ctx)
	}(wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		a.monitor.RunReporter(ctx)
	}(wg)

	// Graceful shutdown by OS signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	a.log.Sugar().Infof("Gracefully shutting down agent...")

	// Cancel the context to stop goroutines.
	cancel()

	// Waiting for goroutines to finish.
	wg.Wait()

	return nil
}
