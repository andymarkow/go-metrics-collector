package storage

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/andymarkow/go-metrics-collector/internal/monitor"
)

var _ Storage = (*MemStorage)(nil)

type Metric struct {
	Type  monitor.MetricType
	Value any
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

type CounterValue struct {
	Value int64
}

func (v *CounterValue) String() string {
	return strconv.FormatInt(v.Value, 10)
}

type GaugeValue struct {
	Value float64
}

func (v *GaugeValue) String() string {
	return strconv.FormatFloat(v.Value, 'f', -1, 64)
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

func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric, ok := s.data[name]; ok {
		if v, ok := metric.Value.(CounterValue); ok {
			return v.Value, nil
		}

		return 0, ErrMetricIsNotCounter
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetCounter(name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if metric, ok := s.data[name]; ok {
		if v, ok := metric.Value.(CounterValue); ok {
			s.data[name] = Metric{
				Type:  monitor.MetricCounter,
				Value: CounterValue{Value: v.Value + value},
			}

			return nil
		}

		return ErrMetricIsNotCounter
	}

	s.data[name] = Metric{
		Type:  monitor.MetricCounter,
		Value: CounterValue{Value: value},
	}

	return nil
}

func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric, ok := s.data[name]; ok {
		if v, ok := metric.Value.(GaugeValue); ok {
			return v.Value, nil
		}

		return 0, ErrMetricIsNotGauge
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetGauge(name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if metric, ok := s.data[name]; ok {
		if _, ok := metric.Value.(GaugeValue); !ok {
			return ErrMetricIsNotGauge
		}
	}

	s.data[name] = Metric{
		Type:  monitor.MetricGauge,
		Value: GaugeValue{Value: value},
	}

	return nil
}

func (s *MemStorage) GetAllMetrics() map[string]Metric {
	allMetrics := make(map[string]Metric)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for key, val := range s.data {
		allMetrics[key] = val
	}

	return allMetrics
}
