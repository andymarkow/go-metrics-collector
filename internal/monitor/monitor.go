package monitor

import (
	"fmt"
	"log"
	"runtime"

	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
)

type Collector interface {
	Collect()
	GetName() string
	GetKind() string
	GetValueString() string
}

type Monitor struct {
	client  *httpclient.HTTPClient
	memstat *runtime.MemStats
	metrics []Collector
}

func NewMonitor() *Monitor {
	var memstat runtime.MemStats

	metrics := make([]Collector, 0)

	metrics = append(metrics,
		NewAllocMetric(&memstat),
		NewBuckHashSysMetric(&memstat),
		NewFreesMetric(&memstat),
		NewGCCPUFractionMetric(&memstat),
		NewGCSysMetric(&memstat),
		NewHeapAllocMetric(&memstat),
		NewHeapIdleMetric(&memstat),
		NewHeapInuseMetric(&memstat),
		NewHeapObjectsMetric(&memstat),
		NewHeapReleasedMetric(&memstat),
		NewHeapSysMetric(&memstat),
		NewLastGCMetric(&memstat),
		NewLookupsMetric(&memstat),
		NewMCacheInuseMetric(&memstat),
		NewMCacheSysMetric(&memstat),
		NewMSpanInuseMetric(&memstat),
		NewMSpanSysMetric(&memstat),
		NewMallocsMetric(&memstat),
		NewNextGCMetric(&memstat),
		NewNumForcedGCMetric(&memstat),
		NewNumGCMetric(&memstat),
		NewOtherSysMetric(&memstat),
		NewPauseTotalNsMetric(&memstat),
		NewStackInuseMetric(&memstat),
		NewStackSysMetric(&memstat),
		NewSysMetric(&memstat),
		NewTotalAllocMetric(&memstat),
		NewRandomValueMetric(),
		NewPollCountMetric(),
	)

	client := httpclient.NewHTTPClient()
	client.SetBaseURL("http://localhost:8080")

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
		fmt.Println(v.GetName(), v.GetKind(), v.GetValueString())

		resp, err := m.client.R().SetPathParams(map[string]string{
			"metricType":  v.GetKind(),
			"metricName":  v.GetName(),
			"metricValue": v.GetValueString(),
		}).SetHeader("Content-Type", "text/plain").
			Post("/update/{metricType}/{metricName}/{metricValue}")

		if err != nil {
			log.Println("client.Request:", err)

			continue
		}

		fmt.Println(resp.Status())
	}
}
