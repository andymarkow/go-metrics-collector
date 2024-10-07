// Package reporter provides a metric reporter implementation.
package reporter

import (
	"context"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	grpcclient "github.com/andymarkow/go-metrics-collector/internal/grpc/client/metric/v1"
	"github.com/andymarkow/go-metrics-collector/internal/metricclient"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
	"github.com/andymarkow/go-metrics-collector/internal/signature"
)

// MetricReporter represents a metric reporter.
type MetricReporter struct {
	cfg          *config
	log          *zap.Logger
	metricsChan  <-chan metrics.Metric
	metricClient *metricclient.MetricClient
	grpcClient   *grpcclient.Client
	sendTimeout  time.Duration
	useGrpc      bool
}

type config struct {
	serverAddr  string
	signKey     []byte
	cryptoKey   *rsa.PublicKey
	rateLimiter *rate.Limiter
}

// NewMetricReporter creates a new metric reporter.
func NewMetricReporter(opts ...Option) *MetricReporter {
	rep := &MetricReporter{
		log:         zap.NewNop(),
		cfg:         defaultConfig(),
		sendTimeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(rep)
	}

	if rep.useGrpc {
		grpcClient, err := grpcclient.NewClient(
			grpcclient.WithServerAddr(rep.cfg.serverAddr),
			grpcclient.WithLogger(rep.log),
			grpcclient.WithRateLimiter(rep.cfg.rateLimiter),
		)
		if err != nil {
			rep.log.Error("grpcclient.NewClient", zap.Error(err))
		}

		rep.grpcClient = grpcClient
	} else {
		metricClient := metricclient.NewMetricClient(
			metricclient.WithServerAddr(rep.cfg.serverAddr),
			metricclient.WithLogger(rep.log),
			metricclient.WithRateLimiter(rep.cfg.rateLimiter),
			metricclient.WithCryptoKey(rep.cfg.cryptoKey),
		)

		rep.metricClient = metricClient
	}

	return rep
}

// defaultConfig returns the default config for the metric reporter.
func defaultConfig() *config {
	return &config{
		serverAddr:  "localhost:8080",
		rateLimiter: rate.NewLimiter(rate.Limit(10), 10),
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
		r.useGrpc = useGrpc
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

// WithRateLimiter sets the rate limit for the metric reporter.
func WithRateLimiter(limiter *rate.Limiter) Option {
	return func(r *MetricReporter) {
		r.cfg.rateLimiter = limiter
	}
}

// WithServerAddr sets the server address for the metric reporter.
func WithServerAddr(addr string) Option {
	return func(r *MetricReporter) {
		r.cfg.serverAddr = addr
	}
}

func WithSendTimeout(timeout time.Duration) Option {
	return func(r *MetricReporter) {
		r.sendTimeout = timeout
	}
}

// RegisterSource registers a metrics channel for the metric reporter.
func (r *MetricReporter) RegisterSource(metricsChan <-chan metrics.Metric) {
	r.metricsChan = metricsChan
}

// Run starts the metric reporter.
func (r *MetricReporter) Run() {
	r.log.Info("Starting metric reporter")
	r.log.Info("Reporting method", zap.Bool("grpc", r.useGrpc), zap.Bool("http", !r.useGrpc))

	r.runConsumer()

	r.log.Info("Stopping metric reporter")
}

// runConsumer runs the metrics consumer.
func (r *MetricReporter) runConsumer() {
	const batchSize int = 10

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
	ctx, cancel := context.WithTimeout(context.Background(), r.sendTimeout)
	defer cancel()

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

	// Check if use grpc flag set to true.
	if r.useGrpc {
		r.log.Debug("sending metrics batch by grpc")

		if err := r.sendByGRPC(ctx, hashsum, payload); err != nil {
			return fmt.Errorf("sendByGRPC: %w", err)
		}

		return nil
	}

	r.log.Debug("sending metrics batch by http")

	if err := r.sendByHTTP(ctx, hashsum, payload); err != nil {
		return fmt.Errorf("sendByHTTP: %w", err)
	}

	return nil
}

// sendByHTTP sends a request to the server by http.
func (r *MetricReporter) sendByHTTP(ctx context.Context, hashsum string, payload []byte) error {
	err := r.metricClient.UpdateMetrics(ctx, hashsum, payload)
	if err != nil {
		return fmt.Errorf("metricClient.UpdateMetrics: %w", err)
	}

	return nil
}

func (r *MetricReporter) sendByGRPC(ctx context.Context, hashsum string, payload []byte) error {
	msg, err := r.grpcClient.UpdateMetricsV1(ctx, hashsum, payload)
	if err != nil {
		return fmt.Errorf("grpcClient.UpdateMetricsV1: %w", err)
	}

	r.log.Info("grpc response", zap.Any("status", msg))

	return nil
}
