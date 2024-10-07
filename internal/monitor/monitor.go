// Package monitor provides a metrics monitor.
package monitor

import (
	"context"
	"crypto/rsa"
	"runtime"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/andymarkow/go-metrics-collector/internal/monitor/collector"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/reporter"
)

// Monitor is a metrics monitor.
type Monitor struct {
	cfg       *config
	log       *zap.Logger
	collector *collector.MetricCollector
	reporter  *reporter.MetricReporter
}

// config represents the monitor configuration.
type config struct {
	useGrpc        bool
	cryptoPubKey   *rsa.PublicKey
	signKey        []byte
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
	serverAddr     string
}

// NewMonitor creates a new Monitor with the given options.
func NewMonitor(opts ...Option) *Monitor {
	mon := &Monitor{
		cfg: defaultConfig(),
		log: zap.Must(zap.NewDevelopment()),
	}

	for _, opt := range opts {
		opt(mon)
	}

	col := collector.NewCollector(
		collector.WithLogger(mon.log),
		collector.WithPollInterval(mon.cfg.pollInterval),
	)

	col.Register(getMetrics()...)

	mon.collector = col

	mon.reporter = reporter.NewMetricReporter(
		reporter.WithLogger(mon.log),
		reporter.WithServerAddr(mon.cfg.serverAddr),
		reporter.WithSignKey(mon.cfg.signKey),
		reporter.WithCryptoKey(mon.cfg.cryptoPubKey),
		reporter.WithRateLimit(mon.cfg.rateLimit),
		reporter.WithUseGrpc(mon.cfg.useGrpc),
	)

	return mon
}

func defaultConfig() *config {
	return &config{
		pollInterval:   5 * time.Second,
		reportInterval: 5 * time.Second,
		rateLimit:      10,
	}
}

// Option is a monitor option.
type Option func(m *Monitor)

// WithLogger is a monitor option that sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(m *Monitor) {
		m.log = logger
	}
}

// WithServerAddr is a monitor option that sets server address.
func WithServerAddr(addr string) Option {
	return func(m *Monitor) {
		m.cfg.serverAddr = addr
	}
}

// WithSignKey is a monitor option that sets sign key.
func WithSignKey(signKey []byte) Option {
	return func(m *Monitor) {
		m.cfg.signKey = signKey
	}
}

// WithCryptoPubKey is a monitor option that sets crypto public key.
func WithCryptoPubKey(cryptoPubKey *rsa.PublicKey) Option {
	return func(m *Monitor) {
		m.cfg.cryptoPubKey = cryptoPubKey
	}
}

// WithPollInterval is a monitor option that sets poll interval.
func WithPollInterval(pollInterval time.Duration) Option {
	return func(m *Monitor) {
		m.cfg.pollInterval = pollInterval
	}
}

// WithReportInterval is a monitor option that sets report interval.
func WithReportInterval(reportInterval time.Duration) Option {
	return func(m *Monitor) {
		m.cfg.reportInterval = reportInterval
	}
}

// WithRateLimit is a monitor option that sets rate limit.
func WithRateLimit(rateLimit int) Option {
	return func(m *Monitor) {
		m.cfg.rateLimit = rateLimit
	}
}

// WithUseGrpc is a monitor option that sets use grpc flag.
func WithUseGrpc(useGrpc bool) Option {
	return func(m *Monitor) {
		m.cfg.useGrpc = useGrpc
	}
}

// Run runs the metrics monitor.
func (m *Monitor) Run(ctx context.Context) {
	m.log.Info("Starting metrics monitor")

	errgrp, grpCtx := errgroup.WithContext(ctx)

	metricsChan := m.collector.RunProducer(grpCtx)

	m.reporter.RegisterSource(metricsChan)

	errgrp.Go(func() error {
		m.collector.RunCollector(grpCtx)

		return nil
	})

	errgrp.Go(func() error { //nolint:contextcheck
		m.reporter.Run()

		return nil
	})

	if err := errgrp.Wait(); err != nil {
		m.log.Error("failed to stop metrics monitor", zap.Error(err))
	}

	m.log.Info("Stopping metrics monitor")
}

func getMetrics() []metrics.Metric {
	var memstat runtime.MemStats

	return []metrics.Metric{
		metrics.NewAllocMetric(&memstat),
		metrics.NewBuckHashSysMetric(&memstat),
		metrics.NewFreesMetric(&memstat),
		metrics.NewGCCPUFractionMetric(&memstat),
		metrics.NewGCSysMetric(&memstat),
		metrics.NewHeapAllocMetric(&memstat),
		metrics.NewHeapIdleMetric(&memstat),
		metrics.NewHeapInuseMetric(&memstat),
		metrics.NewHeapObjectsMetric(&memstat),
		metrics.NewHeapReleasedMetric(&memstat),
		metrics.NewHeapSysMetric(&memstat),
		metrics.NewLastGCMetric(&memstat),
		metrics.NewLookupsMetric(&memstat),
		metrics.NewMCacheInuseMetric(&memstat),
		metrics.NewMCacheSysMetric(&memstat),
		metrics.NewMSpanInuseMetric(&memstat),
		metrics.NewMSpanSysMetric(&memstat),
		metrics.NewMallocsMetric(&memstat),
		metrics.NewNextGCMetric(&memstat),
		metrics.NewNumForcedGCMetric(&memstat),
		metrics.NewNumGCMetric(&memstat),
		metrics.NewOtherSysMetric(&memstat),
		metrics.NewPauseTotalNsMetric(&memstat),
		metrics.NewStackInuseMetric(&memstat),
		metrics.NewStackSysMetric(&memstat),
		metrics.NewSysMetric(&memstat),
		metrics.NewTotalAllocMetric(&memstat),
		metrics.NewRandomValueMetric(),
		metrics.NewPollCountMetric(),
		metrics.NewTotalMemoryMetric(),
		metrics.NewFreeMemoryMetric(),
		metrics.NewCPUutilizationMetric(),
	}
}
