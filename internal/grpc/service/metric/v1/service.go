// Package v1 provides a gRPC Metric service implementation.
package v1

import (
	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/storage"

	pbv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/api/metric/v1"
)

type MetricService struct {
	pbv1.UnimplementedMetricServiceServer

	log     *zap.Logger
	storage storage.Storage
}

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

type Option func(*MetricService)

func WithLogger(log *zap.Logger) Option {
	return func(c *MetricService) {
		c.log = log
	}
}
