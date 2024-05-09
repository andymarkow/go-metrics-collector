package monitor

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"go.uber.org/zap"
)

type Metric interface {
	Collect()
	GetName() string
	GetKind() string
	GetValue() any
	GetValueString() string
}

type Reseter interface {
	Reset()
}

type Monitor struct {
	log     *zap.Logger
	client  *httpclient.HTTPClient
	memstat *runtime.MemStats
	metrics []Metric
}

func NewMonitor(opts ...Option) *Monitor {
	var memstat runtime.MemStats

	metrics := make([]Metric, 0)

	metrics = append(metrics,
		newAllocMetric(&memstat),
		newBuckHashSysMetric(&memstat),
		newFreesMetric(&memstat),
		newGCCPUFractionMetric(&memstat),
		newGCSysMetric(&memstat),
		newHeapAllocMetric(&memstat),
		newHeapIdleMetric(&memstat),
		newHeapInuseMetric(&memstat),
		newHeapObjectsMetric(&memstat),
		newHeapReleasedMetric(&memstat),
		newHeapSysMetric(&memstat),
		newLastGCMetric(&memstat),
		newLookupsMetric(&memstat),
		newMCacheInuseMetric(&memstat),
		newMCacheSysMetric(&memstat),
		newMSpanInuseMetric(&memstat),
		newMSpanSysMetric(&memstat),
		newMallocsMetric(&memstat),
		newNextGCMetric(&memstat),
		newNumForcedGCMetric(&memstat),
		newNumGCMetric(&memstat),
		newOtherSysMetric(&memstat),
		newPauseTotalNsMetric(&memstat),
		newStackInuseMetric(&memstat),
		newStackSysMetric(&memstat),
		newSysMetric(&memstat),
		newTotalAllocMetric(&memstat),
		newRandomValueMetric(),
		newPollCountMetric(),
	)

	client := httpclient.NewHTTPClient()

	mon := &Monitor{
		log:     zap.Must(zap.NewDevelopment()),
		client:  client,
		memstat: &memstat,
		metrics: metrics,
	}

	// Apply options
	for _, opt := range opts {
		opt(mon)
	}

	return mon
}

// Option is a monitor option.
type Option func(m *Monitor)

// WithLogger is a monitor option that sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(m *Monitor) {
		m.log = logger
	}
}

func WithServerAddr(addr string) Option {
	return func(m *Monitor) {
		m.client.SetBaseURL(addr)
	}
}

func (m *Monitor) Collect() {
	runtime.ReadMemStats(m.memstat)

	for _, v := range m.metrics {
		v.Collect()
	}
}

func (m *Monitor) Push() {
	var metrics []models.Metrics

	batchSize := 100

	for _, v := range m.metrics {
		switch v.GetKind() {
		case string(MetricCounter):
			val, ok := v.GetValue().(int64)
			if !ok {
				m.log.Error("cant assert type int64: v.GetValue().(int64)")

				continue
			}

			metrics = append(metrics, models.Metrics{
				ID:    v.GetName(),
				MType: v.GetKind(),
				Delta: &val,
			})

		case string(MetricGauge):
			val, ok := v.GetValue().(float64)
			if !ok {
				m.log.Error("cant assert type float64: v.GetValue().(float64)")

				continue
			}

			metrics = append(metrics, models.Metrics{
				ID:    v.GetName(),
				MType: v.GetKind(),
				Value: &val,
			})
		}

		// Batch limit
		if len(metrics) >= batchSize {
			if err := m.sendRequest(metrics); err != nil {
				m.log.Error("sendRequest: " + err.Error())

				continue
			}

			// Flush slice
			metrics = metrics[:0]
		}

		if c, ok := v.(Reseter); ok {
			c.Reset()
		}
	}

	if len(metrics) > 0 {
		if err := m.sendRequest(metrics); err != nil {
			m.log.Error("sendRequest: " + err.Error())
		}
	}
}

func (m *Monitor) sendRequest(metrics []models.Metrics) error {
	payload, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	zbuf := gzip.NewWriter(buf)
	defer zbuf.Close()

	if _, err := zbuf.Write(payload); err != nil {
		return fmt.Errorf("zbuf.Write: %w", err)
	}
	zbuf.Flush()

	_, err = m.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(buf.Bytes()).
		Post("/updates")
	if err != nil {
		return fmt.Errorf("client.Request: %w", err)
	}

	return nil
}
