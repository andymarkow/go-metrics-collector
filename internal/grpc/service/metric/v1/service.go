// Package v1 provides a gRPC Metric service implementation.
package v1

import (
	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/storage"

	pbv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/api/metric/v1"
)

// MetricService represents a gRPC Metric service.
type MetricService struct {
	pbv1.UnimplementedMetricServiceServer

	log     *zap.Logger
	storage storage.Storage
	signKey []byte
}

// NewMetricService returns a new MetricService instance.
func NewMetricService(store storage.Storage, opts ...Option) *MetricService {
	svc := &MetricService{
		log:     zap.NewNop(),
		storage: store,
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

// Option is a functional option type for MetricService.
type Option func(*MetricService)

// WithLogger sets the logger for the Metric service.
func WithLogger(log *zap.Logger) Option {
	return func(c *MetricService) {
		c.log = log
	}
}

// WithSignKey sets the sign key for the Metric service.
func WithSignKey(signKey []byte) Option {
	return func(c *MetricService) {
		c.signKey = signKey
	}
}
