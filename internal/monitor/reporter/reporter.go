// Package reporter provides a metric reporter implementation.
package reporter

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/andymarkow/go-metrics-collector/internal/cryptutils"
	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
	"github.com/andymarkow/go-metrics-collector/internal/signature"
)

// MetricReporter represents a metric reporter.
type MetricReporter struct {
	cfg         *config
	log         *zap.Logger
	httpClient  *httpclient.HTTPClient
	metricsChan <-chan metrics.Metric
	rateLimiter *rate.Limiter
}

type config struct {
	serverAddr string
	useGrpc    bool
	signKey    []byte
	cryptoKey  *rsa.PublicKey
}

// NewMetricReporter creates a new metric reporter.
func NewMetricReporter(opts ...Option) *MetricReporter {
	reporter := &MetricReporter{
		log:         zap.NewNop(),
		httpClient:  httpclient.NewHTTPClient(),
		cfg:         defaultConfig(),
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 10),
	}

	for _, opt := range opts {
		opt(reporter)
	}

	reporter.httpClient.SetBaseURL(reporter.cfg.serverAddr)

	return reporter
}

// defaultConfig returns the default config for the metric reporter.
func defaultConfig() *config {
	return &config{
		serverAddr: "localhost:8080",
		useGrpc:    false,
	}
}

// Option is a reporter option.
type Option func(r *MetricReporter)

// WithLogger sets the logger for the metric reporter.
func WithLogger(logger *zap.Logger) Option {
	return func(r *MetricReporter) {
		r.log = logger
	}
}

// WithUseGrpc sets the use grpc flag for the metric reporter.
func WithUseGrpc(useGrpc bool) Option {
	return func(r *MetricReporter) {
		r.cfg.useGrpc = useGrpc
	}
}

// WithSignKey sets the sign key for the metric reporter.
func WithSignKey(signKey []byte) Option {
	return func(r *MetricReporter) {
		r.cfg.signKey = signKey
	}
}

// WithCryptoKey sets the crypto public key for the metric reporter.
func WithCryptoKey(cryptoKey *rsa.PublicKey) Option {
	return func(r *MetricReporter) {
		r.cfg.cryptoKey = cryptoKey
	}
}

// WithRateLimit sets the rate limit for the metric reporter.
func WithRateLimit(limit int) Option {
	return func(r *MetricReporter) {
		r.rateLimiter = rate.NewLimiter(rate.Every(time.Second), limit)
	}
}

// WithServerAddr sets the server address for the metric reporter.
func WithServerAddr(addr string) Option {
	return func(r *MetricReporter) {
		r.cfg.serverAddr = addr
	}
}

// RegisterSource registers a metrics channel for the metric reporter.
func (r *MetricReporter) RegisterSource(metricsChan <-chan metrics.Metric) {
	r.metricsChan = metricsChan
}

// Run starts the metric reporter.
func (r *MetricReporter) Run() {
	r.log.Info("Starting metric reporter")

	r.runConsumer()

	r.log.Info("Stopping metric reporter")
}

// runConsumer runs the metrics consumer.
func (r *MetricReporter) runConsumer() {
	const batchSize int = 50

	r.log.Info("Starting metrics consumer")

	var metricsBatch []models.Metrics

	for metric := range r.metricsChan {
		r.log.Debug("Processing metric",
			zap.String("name", metric.GetName()),
			zap.String("kind", metric.GetKind()),
			zap.Any("value", metric.GetValue()),
		)

		switch metric.GetKind() {
		case string(metrics.MetricCounter):
			val, ok := metric.GetValue().(int64)
			if !ok {
				r.log.Error("cant assert type int64: v.GetValue().(int64)")

				continue
			}

			metricsBatch = append(metricsBatch, models.Metrics{
				ID:    metric.GetName(),
				MType: metric.GetKind(),
				Delta: &val,
			})

		case string(metrics.MetricGauge):
			val, ok := metric.GetValue().(float64)
			if !ok {
				r.log.Error("cant assert type float64: metric.GetValue().(float64)")

				continue
			}

			metricsBatch = append(metricsBatch, models.Metrics{
				ID:    metric.GetName(),
				MType: metric.GetKind(),
				Value: &val,
			})
		}

		// Check if the batch is full.
		if len(metricsBatch) >= batchSize {
			if err := r.sendRequest(metricsBatch); err != nil {
				r.log.Error("failed to process metrics send request", zap.Error(err))

				continue
			}

			// Flush slice.
			metricsBatch = metricsBatch[:0]
		}
	}

	if len(metricsBatch) > 0 {
		if err := r.sendRequest(metricsBatch); err != nil {
			r.log.Error("failed to process metrics send request", zap.Error(err))
		}
	}
}

// sendRequest sends a request to the server.
func (r *MetricReporter) sendRequest(metrics []models.Metrics) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rateLimiter.Wait: %w", err)
	}

	payload, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	var hashsum string

	// Calculate hash sum of the payload with a signature key.
	if len(r.cfg.signKey) > 0 {
		sign, err := signature.CalculateHashSum(r.cfg.signKey, payload)
		if err != nil {
			return fmt.Errorf("signPayload: %w", err)
		}

		r.log.Debug("payload signature", zap.String("hashsum", hex.EncodeToString(sign)))

		hashsum = hex.EncodeToString(sign)
	}

	// TODO: add grpc support
	// if r.useGrpc {
	// }

	if err := r.sendByHTTP(hashsum, payload); err != nil {
		return fmt.Errorf("sendByHTTP: %w", err)
	}

	return nil
}

// sendByHTTP sends a request to the server by http.
func (r *MetricReporter) sendByHTTP(hashsum string, payload []byte) error {
	// Compress payload data with gzip compression method.
	body, err := compressDataGzip(payload)
	if err != nil {
		return fmt.Errorf("failed to compress payload data with gzip: %w", err)
	}

	// If crypto public key is set, encrypt payload data with a public RSA key.
	if r.cfg.cryptoKey != nil {
		// Encrypt payload data with a public RSA key.
		cryptoHash := sha256.New()

		// Encrypt payload data with a public RSA key.
		encryptedBody, err := cryptutils.EncryptOAEP(cryptoHash, rand.Reader, r.cfg.cryptoKey, body, nil)
		if err != nil {
			return fmt.Errorf("cryptutils.EncryptOAEP: %w", err)
		}

		r.log.Debug("encrypted payload content", zap.Any("data", encryptedBody))

		// Set encrypted payload data to the request body.
		body = encryptedBody
	}

	ip, err := getIPAddress()
	if err != nil {
		return fmt.Errorf("failed to get IP address: %w", err)
	}

	req := r.httpClient.R().
		SetHeader("X-Real-IP", ip.String()).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(body)

	if hashsum != "" {
		req.SetHeader("HashSHA256", hashsum)
	}

	// Send payload data to the remote server.
	resp, err := req.Post("/updates")
	if err != nil {
		return fmt.Errorf("client.Request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to send data: %d - %s", resp.StatusCode(), resp.String())
	}

	return nil
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
