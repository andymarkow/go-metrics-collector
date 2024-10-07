package storage

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
)

var _ Storage = (*MemStorage)(nil)

type Metric struct {
	Value any                `json:"value"`
	Type  metrics.MetricType `json:"type"`
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

func (s *MemStorage) GetAllMetrics(_ context.Context) (map[string]Metric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data, nil
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
				Type:  metrics.MetricCounter,
				Value: CounterValue(int64(v) + value),
			}

			return nil
		}

		return ErrMetricIsNotCounter
	}

	s.data[name] = Metric{
		Type:  metrics.MetricCounter,
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
		Type:  metrics.MetricGauge,
		Value: GaugeValue(value),
	}

	return nil
}

func (s *MemStorage) SetMetrics(ctx context.Context, metrics []models.Metrics) error {
	for _, metric := range metrics {
		switch metric.MType {
		case "counter":
			if err := s.SetCounter(ctx, metric.ID, *metric.Delta); err != nil {
				return fmt.Errorf("failed to set metric (%s): %w", metric.ID, err)
			}

		case "gauge":
			if err := s.SetGauge(ctx, metric.ID, *metric.Value); err != nil {
				return fmt.Errorf("failed to set metric (%s): %w", metric.ID, err)
			}
		}
	}

	return nil
}

func (s *MemStorage) LoadData(_ context.Context, data map[string]Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, metric := range data {
		switch metric.Type {
		case metrics.MetricCounter:
			v, ok := metric.Value.(float64)
			if !ok {
				return fmt.Errorf("failed load metric (%s): invalid value type (%T)", k, metric.Value)
			}

			s.data[k] = Metric{
				Type:  metric.Type,
				Value: CounterValue(int64(v)),
			}

		case metrics.MetricGauge:
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
