// Package collector provides a metrics collector implementation.
package collector

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
)

// MetricCollector represents a metrics collector.
type MetricCollector struct {
	log            *zap.Logger
	metrics        []metrics.Metric
	pollInterval   time.Duration
	reportInterval time.Duration
}

// NewCollector creates a new metrics collector.
func NewCollector(opts ...Option) *MetricCollector {
	c := &MetricCollector{
		log:            zap.NewNop(),
		pollInterval:   5 * time.Second,
		reportInterval: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Option represents a metrics collector option.
type Option func(c *MetricCollector)

// WithLogger sets the logger for the collector.
func WithLogger(log *zap.Logger) Option {
	return func(c *MetricCollector) {
		c.log = log
	}
}

// WithPollInterval sets the poll interval for the metrics collector.
func WithPollInterval(interval time.Duration) Option {
	return func(c *MetricCollector) {
		c.pollInterval = interval
	}
}

// WithReportInterval sets the report interval for the metrics collector.
func WithReportInterval(interval time.Duration) Option {
	return func(c *MetricCollector) {
		c.reportInterval = interval
	}
}

// Register registers a metrics collector metrics.
func (c *MetricCollector) Register(m ...metrics.Metric) {
	c.metrics = append(c.metrics, m...)
}

// RunProducer runs the metrics producer.
func (c *MetricCollector) RunProducer(ctx context.Context) chan metrics.Metric {
	metricsChan := make(chan metrics.Metric, len(c.metrics))

	go func() {
		defer close(metricsChan)

		ticker := time.NewTicker(c.reportInterval)
		defer ticker.Stop()

		c.log.Info("Starting metrics producer")

		for {
			select {
			case <-ctx.Done():
				c.log.Info("Stopping metrics producer")

				c.flush(metricsChan)

				return

			case <-ticker.C:
				c.flush(metricsChan)
			}
		}
	}()

	return metricsChan
}

// RunCollector runs the metrics collector.
func (c *MetricCollector) RunCollector(ctx context.Context) {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	c.log.Info("Starting metrics collector")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Stopping metrics collector")

			return

		case <-ticker.C:
			c.log.Info("Collecting metrics")

			c.collect()
		}
	}
}

// collect collects metrics.
func (c *MetricCollector) collect() {
	for _, m := range c.metrics {
		m.Collect()
	}
}

func (c *MetricCollector) flush(metricsChan chan metrics.Metric) {
	for _, m := range c.metrics {
		c.log.Debug("Producing metric",
			zap.String("name", m.GetName()),
			zap.String("kind", m.GetKind()),
			zap.Any("value", m.GetValue()),
		)

		metricsChan <- m

		// Reset counter metric.
		if c, ok := m.(metrics.Reseter); ok {
			c.Reset()
		}
	}
}
