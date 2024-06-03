package monitor

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/signature"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Metric interface {
	Collect()
	GetName() string
	GetKind() string
	GetValue() any
	GetValueString() string
}

type Reseter interface {
	Reset()
}

type Monitor struct {
	log            *zap.Logger
	client         *httpclient.HTTPClient
	memstat        *runtime.MemStats
	metrics        []Metric
	gopsutilstats  []Metric
	signKey        []byte
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
}

func NewMonitor(opts ...Option) *Monitor {
	var memstat runtime.MemStats

	metrics := make([]Metric, 0)

	metrics = append(metrics,
		newAllocMetric(&memstat),
		newBuckHashSysMetric(&memstat),
		newFreesMetric(&memstat),
		newGCCPUFractionMetric(&memstat),
		newGCSysMetric(&memstat),
		newHeapAllocMetric(&memstat),
		newHeapIdleMetric(&memstat),
		newHeapInuseMetric(&memstat),
		newHeapObjectsMetric(&memstat),
		newHeapReleasedMetric(&memstat),
		newHeapSysMetric(&memstat),
		newLastGCMetric(&memstat),
		newLookupsMetric(&memstat),
		newMCacheInuseMetric(&memstat),
		newMCacheSysMetric(&memstat),
		newMSpanInuseMetric(&memstat),
		newMSpanSysMetric(&memstat),
		newMallocsMetric(&memstat),
		newNextGCMetric(&memstat),
		newNumForcedGCMetric(&memstat),
		newNumGCMetric(&memstat),
		newOtherSysMetric(&memstat),
		newPauseTotalNsMetric(&memstat),
		newStackInuseMetric(&memstat),
		newStackSysMetric(&memstat),
		newSysMetric(&memstat),
		newTotalAllocMetric(&memstat),
		newRandomValueMetric(),
		newPollCountMetric(),
	)

	gopsutilstats := make([]Metric, 0)

	gopsutilstats = append(gopsutilstats,
		newTotalMemoryMetric(),
		newFreeMemoryMetric(),
		newCPUutilizationMetric(),
	)

	client := httpclient.NewHTTPClient()

	mon := &Monitor{
		log:           zap.Must(zap.NewDevelopment()),
		client:        client,
		memstat:       &memstat,
		metrics:       metrics,
		gopsutilstats: gopsutilstats,
	}

	// Apply options
	for _, opt := range opts {
		opt(mon)
	}

	// Configure the retry strategy
	client.
		SetLogger(mon.log.Sugar()).
		SetRetryCount(3).                  // Number of retry attempts
		SetRetryWaitTime(1 * time.Second). // Initial wait time between retries
		SetRetryMaxWaitTime(10 * time.Second).
		SetRetryAfter(retryAfterWithInterval(2)).
		AddRetryCondition(func(_ *resty.Response, err error) bool {
			// Retry for retryable errors
			return isRetryableError(err)
		})

	return mon
}

// retryAfterWithInterval returns duration intervals between retries.
func retryAfterWithInterval(retryWaitInterval int) resty.RetryAfterFunc {
	return func(_ *resty.Client, resp *resty.Response) (time.Duration, error) {
		return time.Duration((resp.Request.Attempt*retryWaitInterval - 1)) * time.Second, nil
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

func WithServerAddr(addr string) Option {
	return func(m *Monitor) {
		m.client.SetBaseURL(addr)
	}
}

func WithSignKey(signKey []byte) Option {
	return func(m *Monitor) {
		m.signKey = signKey
	}
}

func WithPollInterval(pollInterval time.Duration) Option {
	return func(m *Monitor) {
		m.pollInterval = pollInterval
	}
}

func WithReportInterval(reportInterval time.Duration) Option {
	return func(m *Monitor) {
		m.reportInterval = reportInterval
	}
}

func WithRateLimit(rateLimit int) Option {
	return func(m *Monitor) {
		m.rateLimit = rateLimit
	}
}

func (m *Monitor) RunCollector(ctx context.Context) {
	pollTicker := time.NewTicker(m.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			m.Collect()
		}
	}
}

func (m *Monitor) RunCollectorGopsutils(ctx context.Context) {
	pollTicker := time.NewTicker(m.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			for _, v := range m.gopsutilstats {
				v.Collect()
			}
		}
	}
}

func (m *Monitor) RunReporter(ctx context.Context) {
	reportTicker := time.NewTicker(m.reportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-reportTicker.C:
			m.ReportMetrics(append(m.metrics, m.gopsutilstats...))
		}
	}
}

// Collect collects metrics.
func (m *Monitor) Collect() {
	runtime.ReadMemStats(m.memstat)

	for _, v := range m.metrics {
		v.Collect()
	}
}

func (m *Monitor) reportWorker(wg *sync.WaitGroup, metricsChan <-chan Metric) {
	defer wg.Done()

	const batchSize int = 100

	var metrics []models.Metrics

	for metric := range metricsChan {
		m.log.Debug("reporting", zap.String("metric", metric.GetName()))

		switch metric.GetKind() {
		case string(MetricCounter):
			val, ok := metric.GetValue().(int64)
			if !ok {
				m.log.Error("cant assert type int64: v.GetValue().(int64)")

				continue
			}

			metrics = append(metrics, models.Metrics{
				ID:    metric.GetName(),
				MType: metric.GetKind(),
				Delta: &val,
			})

		case string(MetricGauge):
			val, ok := metric.GetValue().(float64)
			if !ok {
				m.log.Error("cant assert type float64: metric.GetValue().(float64)")

				continue
			}

			metrics = append(metrics, models.Metrics{
				ID:    metric.GetName(),
				MType: metric.GetKind(),
				Value: &val,
			})
		}

		// Batch size limit
		if len(metrics) >= batchSize {
			if err := m.sendRequest(metrics); err != nil {
				m.log.Error("sendRequest: " + err.Error())

				continue
			}

			// Flush slice
			metrics = metrics[:0]
		}

		// Reset counter metric
		if c, ok := metric.(Reseter); ok {
			c.Reset()
		}
	}

	if len(metrics) > 0 {
		if err := m.sendRequest(metrics); err != nil {
			m.log.Error("sendRequest: " + err.Error())
		}
	}
}

func (m *Monitor) ReportMetrics(metrics []Metric) {
	metricsChan := make(chan Metric, m.rateLimit)

	wg := &sync.WaitGroup{}

	// Spawn workers
	for w := 1; w <= m.rateLimit; w++ {
		wg.Add(1)
		go m.reportWorker(wg, metricsChan)
	}

	// Send metrics to the metrics channel
	for _, v := range metrics {
		metricsChan <- v
	}

	// Close channel and send signal to stop workers
	close(metricsChan)

	wg.Wait()
}

// Report pushes metrics to the remote server.
func (m *Monitor) Report() {
	var metrics []models.Metrics

	batchSize := 100

	for _, v := range m.metrics {
		switch v.GetKind() {
		case string(MetricCounter):
			val, ok := v.GetValue().(int64)
			if !ok {
				m.log.Error("cant assert type int64: v.GetValue().(int64)")

				continue
			}

			metrics = append(metrics, models.Metrics{
				ID:    v.GetName(),
				MType: v.GetKind(),
				Delta: &val,
			})

		case string(MetricGauge):
			val, ok := v.GetValue().(float64)
			if !ok {
				m.log.Error("cant assert type float64: v.GetValue().(float64)")

				continue
			}

			metrics = append(metrics, models.Metrics{
				ID:    v.GetName(),
				MType: v.GetKind(),
				Value: &val,
			})
		}

		// Batch limit
		if len(metrics) >= batchSize {
			if err := m.sendRequest(metrics); err != nil {
				m.log.Error("sendRequest: " + err.Error())

				continue
			}

			// Flush slice
			metrics = metrics[:0]
		}

		if c, ok := v.(Reseter); ok {
			c.Reset()
		}
	}

	if len(metrics) > 0 {
		if err := m.sendRequest(metrics); err != nil {
			m.log.Error("sendRequest: " + err.Error())
		}
	}
}

func (m *Monitor) sendRequest(metrics []models.Metrics) error {
	payload, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	zbuf := gzip.NewWriter(buf)
	defer zbuf.Close()

	if _, err := zbuf.Write(payload); err != nil {
		return fmt.Errorf("zbuf.Write: %w", err)
	}
	zbuf.Flush()

	body := buf.Bytes()

	if len(m.signKey) > 0 {
		sign, err := signature.CalculateHashSum(m.signKey, payload)
		if err != nil {
			return fmt.Errorf("signPayload: %w", err)
		}

		m.log.Debug("signanure", zap.String("sign", hex.EncodeToString(sign)))

		m.client.SetHeader("HashSHA256", hex.EncodeToString(sign))
	}

	_, err = m.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(body).
		Post("/updates")
	if err != nil {
		return fmt.Errorf("client.Request: %w", err)
	}

	return nil
}

// isRetryableError checks if the error is a retryable error.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		// Connection refused error
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			// Connection timeout error
			return true
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// DNS error
		return true
	}

	var addrErr *net.AddrError
	if errors.As(err, &addrErr) {
		// Address error
		return true
	}

	// Operational error
	var opErr *net.OpError

	return errors.As(err, &opErr)
}
