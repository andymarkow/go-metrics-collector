package storage

import (
	"context"
	"errors"
)

var (
	ErrMetricNotFound     = errors.New("metric not found")
	ErrMetricIsNotCounter = errors.New("metric is not counter")
	ErrMetricIsNotGauge   = errors.New("metric is not gauge")
)

type Storage interface {
	GetAllMetrics(ctx context.Context) (map[string]Metric, error)
	GetCounter(ctx context.Context, name string) (int64, error)
	SetCounter(ctx context.Context, name string, value int64) error
	GetGauge(ctx context.Context, name string) (float64, error)
	SetGauge(ctx context.Context, name string, value float64) error
	LoadData(ctx context.Context, data map[string]Metric) error
	Ping(ctx context.Context) error
	Close() error
}

func NewStorage(strg Storage) Storage {
	return strg
}
