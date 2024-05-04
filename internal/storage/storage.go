package storage

import "errors"

type Counter int64
type Gauge float64

var (
	ErrMetricNotFound = errors.New("metric not found")
)

type Storage interface {
	GetAllMetrics() map[string]string
	GetCounter(name string) (int64, error)
	SetCounter(name string, value int64)
	GetGauge(name string) (float64, error)
	SetGauge(name string, value float64)
}

func NewStorage(strg Storage) Storage {
	return strg
}
