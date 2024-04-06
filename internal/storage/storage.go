package storage

import "errors"

type Counter int64
type Gauge float64

var (
	ErrMetricNotFound = errors.New("metric not found")
)

type Storage interface {
	GetAllMetrics() map[string]string
	GetCounter(key string) (int64, error)
	SetCounter(key string, value int64)
	GetGauge(key string) (float64, error)
	SetGauge(key string, value float64)
}

func NewStorage(strg Storage) Storage {
	return strg
}
