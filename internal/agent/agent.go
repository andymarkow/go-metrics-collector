package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	// Graceful shutdown by OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	a.log.Sugar().Infof("Gracefully shutting down agent...")

	// cancel the context to stop goroutines
	cancel()

	// waiting for goroutines to finish
	wg.Wait()

	return nil
}
