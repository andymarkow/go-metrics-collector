package storage

type Counter int64
type Gauge float64

type MemStorage struct {
	counters map[string]Counter
	gauges   map[string]Gauge
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		counters: make(map[string]Counter),
		gauges:   make(map[string]Gauge),
	}
}

func (s *MemStorage) SetCounter(key string, value int64) {
	if v, ok := s.counters[key]; ok {
		s.counters[key] = v + Counter(value)
	} else {
		s.counters[key] = Counter(value)
	}
}

func (s *MemStorage) SetGauge(key string, value float64) {
	s.gauges[key] = Gauge(value)
}
