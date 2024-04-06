package storage

import "fmt"

// var _ Storage = (*MemStorage)(nil)

type MemStorage struct {
	counters map[string]int64
	gauges   map[string]float64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (s *MemStorage) GetCounter(key string) (int64, error) {
	if v, ok := s.counters[key]; ok {
		return v, nil
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetCounter(key string, value int64) {
	if v, ok := s.counters[key]; ok {
		s.counters[key] = v + value
	} else {
		s.counters[key] = value
	}
}

func (s *MemStorage) GetGauge(key string) (float64, error) {
	if v, ok := s.gauges[key]; ok {
		return v, nil
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetGauge(key string, value float64) {
	s.gauges[key] = value
}

func (s *MemStorage) GetAllMetrics() map[string]string {
	all := make(map[string]string)

	for k, v := range s.counters {
		all[k] = fmt.Sprintf("%d", v)
	}

	for k, v := range s.gauges {
		all[k] = fmt.Sprintf("%f", v)
	}

	return all
}
