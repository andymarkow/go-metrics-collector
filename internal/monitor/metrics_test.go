package monitor

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	type want struct {
		name  string
		value string
		kind  string
	}

	source := &runtime.MemStats{
		Alloc:         1024,
		BuckHashSys:   1024,
		Frees:         1024,
		GCCPUFraction: 1024,
		GCSys:         1024,
		HeapAlloc:     1024,
		HeapIdle:      1024,
		HeapInuse:     1024,
		HeapObjects:   1024,
		HeapReleased:  1024,
		HeapSys:       1024,
		LastGC:        1024,
		Lookups:       1024,
		MCacheInuse:   1024,
		MCacheSys:     1024,
		MSpanInuse:    1024,
		MSpanSys:      1024,
		Mallocs:       1024,
		NextGC:        1024,
		NumForcedGC:   1024,
		NumGC:         1024,
		OtherSys:      1024,
		PauseTotalNs:  1024,
		StackInuse:    1024,
		StackSys:      1024,
		Sys:           1024,
		TotalAlloc:    1024,
	}

	testCases := []struct {
		name   string
		want   want
		metric metric
	}{
		{
			name:   "Alloc",
			metric: newAllocMetric(source),
			want:   want{name: "Alloc", kind: "gauge", value: "1024"},
		},
		{
			name:   "BuckHashSys",
			metric: newBuckHashSysMetric(source),
			want:   want{name: "BuckHashSys", kind: "gauge", value: "1024"},
		},
		{
			name:   "Frees",
			metric: newFreesMetric(source),
			want:   want{name: "Frees", kind: "gauge", value: "1024"},
		},
		{
			name:   "GCCPUFraction",
			metric: newGCCPUFractionMetric(source),
			want:   want{name: "GCCPUFraction", kind: "gauge", value: "1024"},
		},
		{
			name:   "GCSys",
			metric: newGCSysMetric(source),
			want:   want{name: "GCSys", kind: "gauge", value: "1024"},
		},
		{
			name:   "HeapAlloc",
			metric: newHeapAllocMetric(source),
			want:   want{name: "HeapAlloc", kind: "gauge", value: "1024"},
		},
		{
			name:   "HeapIdle",
			metric: newHeapIdleMetric(source),
			want:   want{name: "HeapIdle", kind: "gauge", value: "1024"},
		},
		{
			name:   "HeapInuse",
			metric: newHeapInuseMetric(source),
			want:   want{name: "HeapInuse", kind: "gauge", value: "1024"},
		},
		{
			name:   "HeapObjects",
			metric: newHeapObjectsMetric(source),
			want:   want{name: "HeapObjects", kind: "gauge", value: "1024"},
		},
		{
			name:   "HeapReleased",
			metric: newHeapReleasedMetric(source),
			want:   want{name: "HeapReleased", kind: "gauge", value: "1024"},
		},
		{
			name:   "HeapSysMetric",
			metric: newHeapSysMetric(source),
			want:   want{name: "HeapSys", kind: "gauge", value: "1024"},
		},
		{
			name:   "LastGC",
			metric: newLastGCMetric(source),
			want:   want{name: "LastGC", kind: "gauge", value: "1024"},
		},
		{
			name:   "Lookups",
			metric: newLookupsMetric(source),
			want:   want{name: "Lookups", kind: "gauge", value: "1024"},
		},
		{
			name:   "MCacheInuse",
			metric: newMCacheInuseMetric(source),
			want:   want{name: "MCacheInuse", kind: "gauge", value: "1024"},
		},
		{
			name:   "MCacheSys",
			metric: newMCacheSysMetric(source),
			want:   want{name: "MCacheSys", kind: "gauge", value: "1024"},
		},
		{
			name:   "MSpanInuse",
			metric: newMSpanInuseMetric(source),
			want:   want{name: "MSpanInuse", kind: "gauge", value: "1024"},
		},
		{
			name:   "MSpanSys",
			metric: newMSpanSysMetric(source),
			want:   want{name: "MSpanSys", kind: "gauge", value: "1024"},
		},
		{
			name:   "Mallocs",
			metric: newMallocsMetric(source),
			want:   want{name: "Mallocs", kind: "gauge", value: "1024"},
		},
		{
			name:   "NextGC",
			metric: newNextGCMetric(source),
			want:   want{name: "NextGC", kind: "gauge", value: "1024"},
		},
		{
			name:   "NumForcedGC",
			metric: newNumForcedGCMetric(source),
			want:   want{name: "NumForcedGC", kind: "gauge", value: "1024"},
		},
		{
			name:   "NumGC",
			metric: newNumGCMetric(source),
			want:   want{name: "NumGC", kind: "gauge", value: "1024"},
		},
		{
			name:   "OtherSys",
			metric: newOtherSysMetric(source),
			want:   want{name: "OtherSys", kind: "gauge", value: "1024"},
		},
		{
			name:   "PauseTotalNs",
			metric: newPauseTotalNsMetric(source),
			want:   want{name: "PauseTotalNs", kind: "gauge", value: "1024"},
		},
		{
			name:   "StackInuse",
			metric: newStackInuseMetric(source),
			want:   want{name: "StackInuse", kind: "gauge", value: "1024"},
		},
		{
			name:   "StackSys",
			metric: newStackSysMetric(source),
			want:   want{name: "StackSys", kind: "gauge", value: "1024"},
		},
		{
			name:   "Sys",
			metric: newSysMetric(source),
			want:   want{name: "Sys", kind: "gauge", value: "1024"},
		},
		{
			name:   "TotalAlloc",
			metric: newTotalAllocMetric(source),
			want:   want{name: "TotalAlloc", kind: "gauge", value: "1024"},
		},
		{
			name:   "PollCount",
			metric: newPollCountMetric(),
			want:   want{name: "PollCount", kind: "counter", value: "1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.metric.Collect()

			assert.Equal(t, tc.want.name, tc.metric.GetName())
			assert.Equal(t, tc.want.kind, tc.metric.GetKind())
			assert.Equal(t, tc.want.value, tc.metric.GetValueString())
		})
	}
}

func TestRandomValueMetric(t *testing.T) {
	type want struct {
		name string
		kind string
	}

	testCases := []struct {
		name string
		want want
	}{
		{"RandomValue", want{name: "RandomValue", kind: "gauge"}},
	}

	metric := newRandomValueMetric()
	metric.Collect()

	var f float64

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want.name, metric.name)
			assert.Equal(t, tc.want.kind, string(metric.kind))
			assert.IsType(t, f, metric.value)
		})
	}
}
