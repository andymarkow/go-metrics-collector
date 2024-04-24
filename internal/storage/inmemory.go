package storage

import (
	"fmt"
	"sync"
)

var _ Storage = (*MemStorage)(nil)

type memCounter struct {
	data map[string]int64
	mu   sync.RWMutex
}

type memGauge struct {
	data map[string]float64
	mu   sync.RWMutex
}

type MemStorage struct {
	counter memCounter
	gauge   memGauge
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		counter: memCounter{data: make(map[string]int64)},
		gauge:   memGauge{data: make(map[string]float64)},
	}
}

func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.counter.mu.RLock()
	defer s.counter.mu.RUnlock()

	if v, ok := s.counter.data[name]; ok {
		return v, nil
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetCounter(name string, value int64) {
	s.counter.mu.Lock()
	defer s.counter.mu.Unlock()

	if v, ok := s.counter.data[name]; ok {
		s.counter.data[name] = v + value
	} else {
		s.counter.data[name] = value
	}
}

func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.gauge.mu.RLock()
	defer s.gauge.mu.RUnlock()

	if v, ok := s.gauge.data[name]; ok {
		return v, nil
	}

	return 0, ErrMetricNotFound
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.gauge.mu.Lock()
	defer s.gauge.mu.Unlock()

	s.gauge.data[name] = value
}

func (s *MemStorage) GetAllMetrics() map[string]string {
	all := make(map[string]string)

	s.counter.mu.RLock()

	for k, v := range s.counter.data {
		all[k] = fmt.Sprintf("%d", v)
	}
	s.counter.mu.RUnlock()

	s.gauge.mu.RLock()

	for k, v := range s.gauge.data {
		all[k] = fmt.Sprintf("%f", v)
	}
	s.gauge.mu.RUnlock()

	return all
}
