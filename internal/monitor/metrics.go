//nolint:nlreturn
package monitor

import (
	"math/rand"
	"runtime"
	"strconv"
	"sync"
)

type MetricKind string

const (
	MetricCounter MetricKind = "counter"
	MetricGauge   MetricKind = "gauge"
)

type baseMetric struct {
	kind MetricKind
	name string
	mu   sync.Mutex
}

func (m *baseMetric) GetName() string {
	return m.name
}

func (m *baseMetric) GetKind() string {
	return string(m.kind)
}

type CounterMetric struct {
	baseMetric
	value int64
}

func newCounterMetric(name string) CounterMetric {
	return CounterMetric{
		baseMetric: baseMetric{
			kind: MetricCounter,
			name: name,
		},
	}
}

func (m *CounterMetric) GetValueString() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return strconv.FormatInt(m.value, 10)
}

func (m *CounterMetric) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value++
}

func (m *CounterMetric) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = 0
}

type GaugeMetric struct {
	baseMetric
	value float64
}

func newGaugeMetric(name string) GaugeMetric {
	return GaugeMetric{
		baseMetric: baseMetric{
			kind: MetricGauge,
			name: name,
		},
	}
}

func (m *GaugeMetric) GetValueString() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return strconv.FormatFloat(m.value, 'f', -1, 64)
}

type MemStatsMetric struct {
	GaugeMetric
	source *runtime.MemStats
}

func newMemStatsMetric(name string, source *runtime.MemStats) MemStatsMetric {
	return MemStatsMetric{
		GaugeMetric: newGaugeMetric(name),
		source:      source,
	}
}

type (
	Alloc         MemStatsMetric
	BuckHashSys   MemStatsMetric
	Frees         MemStatsMetric
	GCCPUFraction MemStatsMetric
	GCSys         MemStatsMetric
	HeapAlloc     MemStatsMetric
	HeapIdle      MemStatsMetric
	HeapInuse     MemStatsMetric
	HeapObjects   MemStatsMetric
	HeapReleased  MemStatsMetric
	HeapSys       MemStatsMetric
	LastGC        MemStatsMetric
	Lookups       MemStatsMetric
	MCacheInuse   MemStatsMetric
	MCacheSys     MemStatsMetric
	MSpanInuse    MemStatsMetric
	MSpanSys      MemStatsMetric
	Mallocs       MemStatsMetric
	NextGC        MemStatsMetric
	NumForcedGC   MemStatsMetric
	NumGC         MemStatsMetric
	OtherSys      MemStatsMetric
	PauseTotalNs  MemStatsMetric
	StackInuse    MemStatsMetric
	StackSys      MemStatsMetric
	Sys           MemStatsMetric
	TotalAlloc    MemStatsMetric

	RandomValue struct {
		GaugeMetric
	}

	PollCount struct {
		CounterMetric
	}
)

func newAllocMetric(source *runtime.MemStats) *Alloc {
	m := Alloc(newMemStatsMetric("Alloc", source))
	return &m
}

func (m *Alloc) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.Alloc)
}

func newBuckHashSysMetric(source *runtime.MemStats) *BuckHashSys {
	m := BuckHashSys(newMemStatsMetric("BuckHashSys", source))
	return &m
}

func (m *BuckHashSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.BuckHashSys)
}

func newFreesMetric(source *runtime.MemStats) *Frees {
	m := Frees(newMemStatsMetric("Frees", source))
	return &m
}

func (m *Frees) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.Frees)
}

func newGCCPUFractionMetric(source *runtime.MemStats) *GCCPUFraction {
	m := GCCPUFraction(newMemStatsMetric("GCCPUFraction", source))
	return &m
}

func (m *GCCPUFraction) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = m.source.GCCPUFraction
}

func newGCSysMetric(source *runtime.MemStats) *GCSys {
	m := GCSys(newMemStatsMetric("GCSys", source))
	return &m
}

func (m *GCSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.GCSys)
}

func newHeapAllocMetric(source *runtime.MemStats) *HeapAlloc {
	m := HeapAlloc(newMemStatsMetric("HeapAlloc", source))
	return &m
}

func (m *HeapAlloc) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.HeapAlloc)
}

func newHeapIdleMetric(source *runtime.MemStats) *HeapIdle {
	m := HeapIdle(newMemStatsMetric("HeapIdle", source))
	return &m
}

func (m *HeapIdle) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.HeapIdle)
}

func newHeapInuseMetric(source *runtime.MemStats) *HeapInuse {
	m := HeapInuse(newMemStatsMetric("HeapInuse", source))
	return &m
}

func (m *HeapInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.HeapInuse)
}

func newHeapObjectsMetric(source *runtime.MemStats) *HeapObjects {
	m := HeapObjects(newMemStatsMetric("HeapObjects", source))
	return &m
}

func (m *HeapObjects) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.HeapObjects)
}

func newHeapReleasedMetric(source *runtime.MemStats) *HeapReleased {
	m := HeapReleased(newMemStatsMetric("HeapReleased", source))
	return &m
}

func (m *HeapReleased) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.HeapReleased)
}

func newHeapSysMetric(source *runtime.MemStats) *HeapSys {
	m := HeapSys(newMemStatsMetric("HeapSys", source))
	return &m
}

func (m *HeapSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.HeapSys)
}

func newLastGCMetric(source *runtime.MemStats) *LastGC {
	m := LastGC(newMemStatsMetric("LastGC", source))
	return &m
}

func (m *LastGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.LastGC)
}

func newLookupsMetric(source *runtime.MemStats) *Lookups {
	m := Lookups(newMemStatsMetric("Lookups", source))
	return &m
}

func (m *Lookups) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.Lookups)
}

func newMCacheInuseMetric(source *runtime.MemStats) *MCacheInuse {
	m := MCacheInuse(newMemStatsMetric("MCacheInuse", source))
	return &m
}

func (m *MCacheInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.MCacheInuse)
}

func newMCacheSysMetric(source *runtime.MemStats) *MCacheSys {
	m := MCacheSys(newMemStatsMetric("MCacheSys", source))
	return &m
}

func (m *MCacheSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.MCacheSys)
}

func newMSpanInuseMetric(source *runtime.MemStats) *MSpanInuse {
	m := MSpanInuse(newMemStatsMetric("MSpanInuse", source))
	return &m
}

func (m *MSpanInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.MSpanInuse)
}

func newMSpanSysMetric(source *runtime.MemStats) *MSpanSys {
	m := MSpanSys(newMemStatsMetric("MSpanSys", source))
	return &m
}

func (m *MSpanSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.MSpanSys)
}

func newMallocsMetric(source *runtime.MemStats) *Mallocs {
	m := Mallocs(newMemStatsMetric("Mallocs", source))
	return &m
}

func (m *Mallocs) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.Mallocs)
}

func newNextGCMetric(source *runtime.MemStats) *NextGC {
	m := NextGC(newMemStatsMetric("NextGC", source))
	return &m
}

func (m *NextGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.NextGC)
}

func newNumForcedGCMetric(source *runtime.MemStats) *NumForcedGC {
	m := NumForcedGC(newMemStatsMetric("NumForcedGC", source))
	return &m
}

func (m *NumForcedGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.NumForcedGC)
}

func newNumGCMetric(source *runtime.MemStats) *NumGC {
	m := NumGC(newMemStatsMetric("NumGC", source))
	return &m
}

func (m *NumGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.NumGC)
}

func newOtherSysMetric(source *runtime.MemStats) *OtherSys {
	m := OtherSys(newMemStatsMetric("OtherSys", source))
	return &m
}

func (m *OtherSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.OtherSys)
}

func newPauseTotalNsMetric(source *runtime.MemStats) *PauseTotalNs {
	m := PauseTotalNs(newMemStatsMetric("PauseTotalNs", source))
	return &m
}

func (m *PauseTotalNs) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.PauseTotalNs)
}

func newStackInuseMetric(source *runtime.MemStats) *StackInuse {
	m := StackInuse(newMemStatsMetric("StackInuse", source))
	return &m
}

func (m *StackInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.StackInuse)
}

func newStackSysMetric(source *runtime.MemStats) *StackSys {
	m := StackSys(newMemStatsMetric("StackSys", source))
	return &m
}

func (m *StackSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.StackSys)
}

func newSysMetric(source *runtime.MemStats) *Sys {
	m := Sys(newMemStatsMetric("Sys", source))
	return &m
}

func (m *Sys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.Sys)
}

func newTotalAllocMetric(source *runtime.MemStats) *TotalAlloc {
	m := TotalAlloc(newMemStatsMetric("TotalAlloc", source))
	return &m
}

func (m *TotalAlloc) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = float64(m.source.TotalAlloc)
}

func newRandomValueMetric() *RandomValue {
	return &RandomValue{
		GaugeMetric: newGaugeMetric("RandomValue"),
	}
}

func (m *RandomValue) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = rand.Float64() //nolint:gosec
}

func newPollCountMetric() *PollCount {
	return &PollCount{
		CounterMetric: newCounterMetric("PollCount"),
	}
}
