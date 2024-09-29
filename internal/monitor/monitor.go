// Package monitor provides a metrics monitor.
package monitor

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/cryptutils"
	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/signature"
)

// Metric is an interface for metrics.
type Metric interface {
	Collect()
	GetName() string
	GetKind() string
	GetValue() any
	GetValueString() string
}

// Reseter is an interface for metrics that can be reset.
type Reseter interface {
	Reset()
}

// Monitor is a metrics monitor.
type Monitor struct {
	log            *zap.Logger
	client         *httpclient.HTTPClient
	memstat        *runtime.MemStats
	cryptoPubKey   *rsa.PublicKey
	signKey        []byte
	metrics        []Metric
	gopsutilstats  []Metric
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
}

// NewMonitor creates a new Monitor with the given options.
//
// The Monitor is configured with the following metrics by default:
//
//   - Alloc: The number of bytes allocated and still in use.
//   - BuckHashSys: The total size of the hash table used by the runtime.
//   - Frees: The total number of frees.
//   - GCCPUFraction: The fraction of CPU time spent in garbage collection.
//   - GCSys: The total size of memory allocated by the garbage collector.
//   - HeapAlloc: The number of bytes allocated and still in use.
//   - HeapIdle: The number of bytes in idle spans.
//   - HeapInuse: The number of bytes in in-use spans.
//   - HeapObjects: The total number of objects.
//   - HeapReleased: The number of bytes released to the OS.
//   - HeapSys: The total size of the heap.
//   - LastGC: The time of the last garbage collection.
//   - Lookups: The total number of pointer lookups.
//   - MCacheInuse: The number of bytes of mspan structures used by the runtime.
//   - MCacheSys: The total size of memory allocated by the runtime for mspan
//     structures.
//   - MSpanInuse: The number of bytes of mspan structures used by the runtime.
//   - MSpanSys: The total size of memory allocated by the runtime for mspan
//     structures.
//   - Mallocs: The total number of mallocs.
//   - NextGC: The target heap size of the next garbage collection.
//   - NumForcedGC: The total number of forced garbage collections.
//   - NumGC: The total number of garbage collections.
//   - OtherSys: The total size of memory allocated by the runtime for miscellaneous
//     objects.
//   - PauseTotalNs: The total pause time of all garbage collections.
//   - PollCount: The total number of polls.
//   - RandomValue: A random value between 0 and 1, sampled every second.
//   - StackInuse: The number of bytes in use by the stack.
//   - StackSys: The total size of the stack.
//   - Sys: The total size of memory allocated by the runtime.
//   - TotalAlloc: The total number of bytes allocated.
//   - CPUutilization: The CPU utilization of the system.
//   - FreeMemory: The amount of free memory on the system.
//   - TotalMemory: The total amount of memory on the system.
//
// The Monitor also has the following options:
//
//   - HTTP client: The Monitor uses a custom HTTP client with a retry strategy
//     that retries 3 times with exponential backoff.
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

	// Apply options.
	for _, opt := range opts {
		opt(mon)
	}

	// Configure the retry strategy.
	client.
		SetLogger(mon.log.Sugar()).
		SetRetryCount(3).                  // Number of retry attempts
		SetRetryWaitTime(1 * time.Second). // Initial wait time between retries
		SetRetryMaxWaitTime(10 * time.Second).
		SetRetryAfter(retryAfterWithInterval(2)).
		AddRetryCondition(func(_ *resty.Response, err error) bool {
			// Retry for retryable errors.
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

// WithServerAddr is a monitor option that sets server address.
func WithServerAddr(addr string) Option {
	return func(m *Monitor) {
		m.client.SetBaseURL(addr)
	}
}

// WithSignKey is a monitor option that sets sign key.
func WithSignKey(signKey []byte) Option {
	return func(m *Monitor) {
		m.signKey = signKey
	}
}

// WithCryptoPubKey is a monitor option that sets crypto public key.
func WithCryptoPubKey(cryptoPubKey *rsa.PublicKey) Option {
	return func(m *Monitor) {
		m.cryptoPubKey = cryptoPubKey
	}
}

// WithPollInterval is a monitor option that sets poll interval.
func WithPollInterval(pollInterval time.Duration) Option {
	return func(m *Monitor) {
		m.pollInterval = pollInterval
	}
}

// WithReportInterval is a monitor option that sets report interval.
func WithReportInterval(reportInterval time.Duration) Option {
	return func(m *Monitor) {
		m.reportInterval = reportInterval
	}
}

// WithRateLimit is a monitor option that sets rate limit.
func WithRateLimit(rateLimit int) Option {
	return func(m *Monitor) {
		m.rateLimit = rateLimit
	}
}

// RunCollector runs the collector.
func (m *Monitor) RunCollector(ctx context.Context) {
	pollTicker := time.NewTicker(m.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("Stopping metrics collector")

			return

		case <-pollTicker.C:
			m.collect()
		}
	}
}

// RunCollectorGopsutils runs the collector.
func (m *Monitor) RunCollectorGopsutils(ctx context.Context) {
	pollTicker := time.NewTicker(m.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("Stopping gopsutil metrics collector")

			return

		case <-pollTicker.C:
			for _, v := range m.gopsutilstats {
				v.Collect()
			}
		}
	}
}

// RunReporter runs the reporter.
//
// It starts a ticker that triggers every reportInterval.
// When the ticker triggers, it calls ReportMetrics with the metrics
// from the monitor and the gopsutil metrics.
func (m *Monitor) RunReporter(ctx context.Context) {
	reportTicker := time.NewTicker(m.reportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("Stopping metrics reporter")
			m.log.Info("Flushing metrics to remote server")

			m.reportMetrics(append(m.metrics, m.gopsutilstats...))

			return

		case <-reportTicker.C:
			m.reportMetrics(append(m.metrics, m.gopsutilstats...))
		}
	}
}

// Collect collects metrics.
func (m *Monitor) collect() {
	runtime.ReadMemStats(m.memstat)

	for _, v := range m.metrics {
		v.Collect()
	}
}

// ReportMetrics pushes metrics to the remote server.
func (m *Monitor) reportMetrics(metrics []Metric) {
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

// reportWorker sends metrics to the remote server.
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

// sendRequest sends metrics to the remote server.
func (m *Monitor) sendRequest(metrics []models.Metrics) error {
	payload, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	// Calculate hash sum of the payload with a signature key.
	if len(m.signKey) > 0 {
		sign, err := signature.CalculateHashSum(m.signKey, payload)
		if err != nil {
			return fmt.Errorf("signPayload: %w", err)
		}

		m.log.Debug("payload signature", zap.String("hashsum", hex.EncodeToString(sign)))

		m.client.SetHeader("HashSHA256", hex.EncodeToString(sign))
	}

	// Compress payload data with gzip compression method.
	body, err := compressDataGzip(payload)
	if err != nil {
		return fmt.Errorf("failed to compress payload data with gzip: %w", err)
	}

	// If crypto public key is set, encrypt payload data with a public RSA key.
	if m.cryptoPubKey != nil {
		// Encrypt payload data with a public RSA key.
		cryptoHash := sha256.New()

		// Encrypt payload data with a public RSA key.
		encryptedBody, err := cryptutils.EncryptOAEP(cryptoHash, rand.Reader, m.cryptoPubKey, body, nil)
		if err != nil {
			return fmt.Errorf("cryptutils.EncryptOAEP: %w", err)
		}

		m.log.Debug("encrypted payload content", zap.Any("data", encryptedBody))

		// Set encrypted payload data to the request body.
		body = encryptedBody
	}

	ip, err := getIPAddress()
	if err != nil {
		return fmt.Errorf("failed to get IP address: %w", err)
	}

	// Send payload data to the remote server.
	resp, err := m.client.R().
		SetHeader("X-Real-IP", ip.String()).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(body).
		Post("/updates")
	if err != nil {
		return fmt.Errorf("client.Request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to send data: %d - %s", resp.StatusCode(), resp.String())
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

// compressDataGzip compresses the given data using gzip.
//
// The function writes the given data to a gzip writer and then closes the writer.
// If any error occurs while writing or closing, the function returns the error.
//
// If no error occurs, the function returns the compressed data as a byte slice.
func compressDataGzip(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	zbuf := gzip.NewWriter(buf)

	if _, err := zbuf.Write(data); err != nil {
		return nil, fmt.Errorf("zbuf.Write: %w", err)
	}

	if err := zbuf.Close(); err != nil {
		return nil, fmt.Errorf("zbuf.Close: %w", err)
	}

	return buf.Bytes(), nil
}

func getIPAddress() (net.IP, error) {
	// Get a addresses list of all network interfaces.
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("net.InterfaceAddrs: %w", err)
	}

	for _, addr := range addrs {
		// Get the IP address network.
		ipNet, ok := addr.(*net.IPNet)
		// If the IP address is IPv4 and not a loopback address.
		if ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
			// Return first valid non-loopback IPv4 address.
			return ipNet.IP, nil
		}
	}

	// If no valid non-loopback IPv4 address is found, return 127.0.0.1.
	return net.IPv4(127, 0, 0, 1), nil
}
