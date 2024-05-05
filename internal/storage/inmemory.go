package storage

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/andymarkow/go-metrics-collector/internal/monitor"
)

var _ Storage = (*MemStorage)(nil)

type Metric struct {
	Type  monitor.MetricType `json:"type"`
	Value any                `json:"value"`
}

func (m *Metric) StringValue() string {
	switch v := m.Value.(type) {
	case CounterValue:
		return v.String()
	case GaugeValue:
		return v.String()
	}

	return fmt.Sprintf("%v", m.Value)
}

type CounterValue int64

func (v CounterValue) String() string {
	return strconv.FormatInt(int64(v), 10)
}

type GaugeValue float64

func (v GaugeValue) String() string {
	return strconv.FormatFloat(float64(v), 'f', -1, 64)
}

type MemStorage struct {
	data map[string]Metric
	mu   sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]Metric),
	}
}

func (s *MemStorage) Close() error {
	return nil
}

func (s *MemStorage) Ping(_ context.Context) error {
	return nil
}

func (s *MemStorage) GetCounter(_ context.Context, name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric, ok := s.data[name]; ok {
		if v, ok := metric.Value.(CounterValue); ok {
			return int64(v), nil
		}

		return 0, ErrMetricIsNotCounter
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetCounter(_ context.Context, name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if metric, ok := s.data[name]; ok {
		if v, ok := metric.Value.(CounterValue); ok {
			s.data[name] = Metric{
				Type:  monitor.MetricCounter,
				Value: CounterValue(int64(v) + value),
			}

			return nil
		}

		return ErrMetricIsNotCounter
	}

	s.data[name] = Metric{
		Type:  monitor.MetricCounter,
		Value: CounterValue(value),
	}

	return nil
}

func (s *MemStorage) GetGauge(_ context.Context, name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric, ok := s.data[name]; ok {
		if v, ok := metric.Value.(GaugeValue); ok {
			return float64(v), nil
		}

		return 0, ErrMetricIsNotGauge
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetGauge(_ context.Context, name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if metric, ok := s.data[name]; ok {
		if _, ok := metric.Value.(GaugeValue); !ok {
			return ErrMetricIsNotGauge
		}
	}

	s.data[name] = Metric{
		Type:  monitor.MetricGauge,
		Value: GaugeValue(value),
	}

	return nil
}

func (s *MemStorage) GetAllMetrics(_ context.Context) map[string]Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data
}

func (s *MemStorage) LoadData(_ context.Context, data map[string]Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, metric := range data {
		switch metric.Type {
		case monitor.MetricCounter:
			v, ok := metric.Value.(float64)
			if !ok {
				return fmt.Errorf("failed load metric (%s): invalid value type (%T)", k, metric.Value)
			}

			s.data[k] = Metric{
				Type:  metric.Type,
				Value: CounterValue(int64(v)),
			}

		case monitor.MetricGauge:
			v, ok := metric.Value.(float64)
			if !ok {
				return fmt.Errorf("failed load metric (%s): invalid value type (%T)", k, metric.Value)
			}

			s.data[k] = Metric{
				Type:  metric.Type,
				Value: GaugeValue(v),
			}

		default:
			return fmt.Errorf("failed load metric (%s): unknown metric type (%s)", k, metric.Type)
		}
	}

	return nil
}
