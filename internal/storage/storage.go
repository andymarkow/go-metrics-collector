package storage

import "errors"

type Counter int64
type Gauge float64

var (
	ErrMetricNotFound     = errors.New("metric not found")
	ErrMetricIsNotCounter = errors.New("metric is not counter")
	ErrMetricIsNotGauge   = errors.New("metric is not gauge")
)

type Storage interface {
	GetAllMetrics() map[string]Metric
	GetCounter(name string) (int64, error)
	SetCounter(name string, value int64) error
	GetGauge(name string) (float64, error)
	SetGauge(name string, value float64) error
}

func NewStorage(strg Storage) Storage {
	return strg
}
