package monitor

import (
	"log"
	"runtime"

	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
)

type metric interface {
	Collect()
	GetName() string
	GetKind() string
	GetValueString() string
}

type reseter interface {
	Reset()
}

type Monitor struct {
	client  *httpclient.HTTPClient
	memstat *runtime.MemStats
	metrics []metric
}

func NewMonitor(serverAddr string) *Monitor {
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
	client.SetBaseURL(serverAddr)

	return &Monitor{
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
			log.Println("client.Request:", err)

			continue
		}

		if c, ok := v.(reseter); ok {
			c.Reset()
		}
	}
}
