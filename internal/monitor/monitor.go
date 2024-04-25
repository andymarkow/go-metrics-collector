package monitor

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"runtime"

	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"go.uber.org/zap"
)

type metric interface {
	Collect()
	GetName() string
	GetKind() string
	GetValue() any
	GetValueString() string
}

type reseter interface {
	Reset()
}

type Monitor struct {
	log     *zap.Logger
	client  *httpclient.HTTPClient
	memstat *runtime.MemStats
	metrics []metric
}

type Config struct {
	ServerAddr string
	Logger     *zap.Logger
}

func NewMonitor(cfg *Config) *Monitor {
	var memstat runtime.MemStats

	metrics := make([]metric, 0)

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
	client.SetBaseURL(cfg.ServerAddr)

	return &Monitor{
		log:     cfg.Logger,
		client:  client,
		memstat: &memstat,
		metrics: metrics,
	}
}

func (m *Monitor) Collect() {
	runtime.ReadMemStats(m.memstat)

	for _, v := range m.metrics {
		v.Collect()
	}
}

func (m *Monitor) Push() {
	for _, v := range m.metrics {
		_, err := m.client.R().SetPathParams(map[string]string{
			"metricType":  v.GetKind(),
			"metricName":  v.GetName(),
			"metricValue": v.GetValueString(),
		}).SetHeader("Content-Type", "text/plain").
			Post("/update/{metricType}/{metricName}/{metricValue}")

		if err != nil {
			m.log.Error("client.Request: " + err.Error())

			continue
		}

		if c, ok := v.(reseter); ok {
			c.Reset()
		}
	}
}

func (m *Monitor) PushJSON() {
	for _, v := range m.metrics {
		var payload models.Metrics

		switch v.GetKind() {
		case string(MetricCounter):
			val := v.GetValue().(int64)

			payload = models.Metrics{
				ID:    v.GetName(),
				MType: v.GetKind(),
				Delta: &val,
			}

		case string(MetricGauge):
			val := v.GetValue().(float64)

			payload = models.Metrics{
				ID:    v.GetName(),
				MType: v.GetKind(),
				Value: &val,
			}
		}

		m.log.Sugar().Debug("payload: ", payload)

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			m.log.Error("json.Marshal: " + err.Error())

			continue
		}

		buf := bytes.NewBuffer(nil)
		zbuf := gzip.NewWriter(buf)
		defer zbuf.Close()

		if _, err := zbuf.Write(jsonPayload); err != nil {
			m.log.Error("zbuf.Write: " + err.Error())

			continue
		}
		zbuf.Flush()

		_, err = m.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(buf.Bytes()).
			Post("/update")

		if err != nil {
			m.log.Error("client.Request: " + err.Error())

			continue
		}

		if c, ok := v.(reseter); ok {
			c.Reset()
		}
	}
}
