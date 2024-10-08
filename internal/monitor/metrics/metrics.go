// Package metrics provides a metrics implementation.
//
//nolint:nlreturn
package metrics

import (
	"math/rand"
	"runtime"
	"strconv"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// Collector is an interface for metrics that can be collected.
type Collector interface {
	Collect()
}

// Reseter is an interface for metrics that can be reset.
type Reseter interface {
	Reset()
}

// Metric is an interface for metrics.
type Metric interface {
	Collector

	GetName() string
	GetKind() string
	GetValue() any
	GetValueString() string
}

type MetricType string

const (
	MetricCounter MetricType = "counter"
	MetricGauge   MetricType = "gauge"
)

type BaseMetric struct {
	kind MetricType
	name string
	mu   sync.Mutex
}

func (m *BaseMetric) GetName() string {
	return m.name
}

func (m *BaseMetric) GetKind() string {
	return string(m.kind)
}

type CounterMetric struct {
	BaseMetric
	value int64
}

func NewCounterMetric(name string) CounterMetric {
	return CounterMetric{
		BaseMetric: BaseMetric{
			kind: MetricCounter,
			name: name,
		},
	}
}

func (m *CounterMetric) GetValue() any {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.value
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
	BaseMetric
	value float64
}

func NewGaugeMetric(name string) GaugeMetric {
	return GaugeMetric{
		BaseMetric: BaseMetric{
			kind: MetricGauge,
			name: name,
		},
	}
}

func (m *GaugeMetric) GetValue() any {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.value
}

func (m *GaugeMetric) GetValueString() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return strconv.FormatFloat(m.value, 'f', -1, 64)
}

type MemStatsMetric struct {
	source *runtime.MemStats
	GaugeMetric
}

func NewMemStatsMetric(name string, source *runtime.MemStats) MemStatsMetric {
	return MemStatsMetric{
		GaugeMetric: NewGaugeMetric(name),
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

	TotalMemory struct {
		GaugeMetric
	}

	FreeMemory struct {
		GaugeMetric
	}

	CPUutilization struct {
		GaugeMetric
	}
)

func NewAllocMetric(source *runtime.MemStats) *Alloc {
	m := Alloc(NewMemStatsMetric("Alloc", source))
	return &m
}

func (m *Alloc) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.Alloc)
}

func NewBuckHashSysMetric(source *runtime.MemStats) *BuckHashSys {
	m := BuckHashSys(NewMemStatsMetric("BuckHashSys", source))
	return &m
}

func (m *BuckHashSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.BuckHashSys)
}

func NewFreesMetric(source *runtime.MemStats) *Frees {
	m := Frees(NewMemStatsMetric("Frees", source))
	return &m
}

func (m *Frees) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.Frees)
}

func NewGCCPUFractionMetric(source *runtime.MemStats) *GCCPUFraction {
	m := GCCPUFraction(NewMemStatsMetric("GCCPUFraction", source))
	return &m
}

func (m *GCCPUFraction) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = m.source.GCCPUFraction
}

func NewGCSysMetric(source *runtime.MemStats) *GCSys {
	m := GCSys(NewMemStatsMetric("GCSys", source))
	return &m
}

func (m *GCSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.GCSys)
}

func NewHeapAllocMetric(source *runtime.MemStats) *HeapAlloc {
	m := HeapAlloc(NewMemStatsMetric("HeapAlloc", source))
	return &m
}

func (m *HeapAlloc) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.HeapAlloc)
}

func NewHeapIdleMetric(source *runtime.MemStats) *HeapIdle {
	m := HeapIdle(NewMemStatsMetric("HeapIdle", source))
	return &m
}

func (m *HeapIdle) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.HeapIdle)
}

func NewHeapInuseMetric(source *runtime.MemStats) *HeapInuse {
	m := HeapInuse(NewMemStatsMetric("HeapInuse", source))
	return &m
}

func (m *HeapInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.HeapInuse)
}

func NewHeapObjectsMetric(source *runtime.MemStats) *HeapObjects {
	m := HeapObjects(NewMemStatsMetric("HeapObjects", source))
	return &m
}

func (m *HeapObjects) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.HeapObjects)
}

func NewHeapReleasedMetric(source *runtime.MemStats) *HeapReleased {
	m := HeapReleased(NewMemStatsMetric("HeapReleased", source))
	return &m
}

func (m *HeapReleased) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.HeapReleased)
}

func NewHeapSysMetric(source *runtime.MemStats) *HeapSys {
	m := HeapSys(NewMemStatsMetric("HeapSys", source))
	return &m
}

func (m *HeapSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.HeapSys)
}

func NewLastGCMetric(source *runtime.MemStats) *LastGC {
	m := LastGC(NewMemStatsMetric("LastGC", source))
	return &m
}

func (m *LastGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.LastGC)
}

func NewLookupsMetric(source *runtime.MemStats) *Lookups {
	m := Lookups(NewMemStatsMetric("Lookups", source))
	return &m
}

func (m *Lookups) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.Lookups)
}

func NewMCacheInuseMetric(source *runtime.MemStats) *MCacheInuse {
	m := MCacheInuse(NewMemStatsMetric("MCacheInuse", source))
	return &m
}

func (m *MCacheInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.MCacheInuse)
}

func NewMCacheSysMetric(source *runtime.MemStats) *MCacheSys {
	m := MCacheSys(NewMemStatsMetric("MCacheSys", source))
	return &m
}

func (m *MCacheSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.MCacheSys)
}

func NewMSpanInuseMetric(source *runtime.MemStats) *MSpanInuse {
	m := MSpanInuse(NewMemStatsMetric("MSpanInuse", source))
	return &m
}

func (m *MSpanInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.MSpanInuse)
}

func NewMSpanSysMetric(source *runtime.MemStats) *MSpanSys {
	m := MSpanSys(NewMemStatsMetric("MSpanSys", source))
	return &m
}

func (m *MSpanSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.MSpanSys)
}

func NewMallocsMetric(source *runtime.MemStats) *Mallocs {
	m := Mallocs(NewMemStatsMetric("Mallocs", source))
	return &m
}

func (m *Mallocs) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.Mallocs)
}

func NewNextGCMetric(source *runtime.MemStats) *NextGC {
	m := NextGC(NewMemStatsMetric("NextGC", source))
	return &m
}

func (m *NextGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.NextGC)
}

func NewNumForcedGCMetric(source *runtime.MemStats) *NumForcedGC {
	m := NumForcedGC(NewMemStatsMetric("NumForcedGC", source))
	return &m
}

func (m *NumForcedGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.NumForcedGC)
}

func NewNumGCMetric(source *runtime.MemStats) *NumGC {
	m := NumGC(NewMemStatsMetric("NumGC", source))
	return &m
}

func (m *NumGC) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.NumGC)
}

func NewOtherSysMetric(source *runtime.MemStats) *OtherSys {
	m := OtherSys(NewMemStatsMetric("OtherSys", source))
	return &m
}

func (m *OtherSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.OtherSys)
}

func NewPauseTotalNsMetric(source *runtime.MemStats) *PauseTotalNs {
	m := PauseTotalNs(NewMemStatsMetric("PauseTotalNs", source))
	return &m
}

func (m *PauseTotalNs) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.PauseTotalNs)
}

func NewStackInuseMetric(source *runtime.MemStats) *StackInuse {
	m := StackInuse(NewMemStatsMetric("StackInuse", source))
	return &m
}

func (m *StackInuse) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.StackInuse)
}

func NewStackSysMetric(source *runtime.MemStats) *StackSys {
	m := StackSys(NewMemStatsMetric("StackSys", source))
	return &m
}

func (m *StackSys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.StackSys)
}

func NewSysMetric(source *runtime.MemStats) *Sys {
	m := Sys(NewMemStatsMetric("Sys", source))
	return &m
}

func (m *Sys) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.Sys)
}

func NewTotalAllocMetric(source *runtime.MemStats) *TotalAlloc {
	m := TotalAlloc(NewMemStatsMetric("TotalAlloc", source))
	return &m
}

func (m *TotalAlloc) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(m.source)

	m.value = float64(m.source.TotalAlloc)
}

func NewRandomValueMetric() *RandomValue {
	return &RandomValue{
		GaugeMetric: NewGaugeMetric("RandomValue"),
	}
}

func (m *RandomValue) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.value = rand.Float64() //nolint:gosec
}

func NewPollCountMetric() *PollCount {
	return &PollCount{
		CounterMetric: NewCounterMetric("PollCount"),
	}
}

func NewTotalMemoryMetric() *TotalMemory {
	return &TotalMemory{
		GaugeMetric: NewGaugeMetric("TotalMemory"),
	}
}

func (m *TotalMemory) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	m.value = float64(v.Total)
}

func NewFreeMemoryMetric() *FreeMemory {
	return &FreeMemory{
		GaugeMetric: NewGaugeMetric("FreeMemory"),
	}
}

func (m *FreeMemory) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	m.value = float64(v.Free)
}

func NewCPUutilizationMetric() *CPUutilization {
	return &CPUutilization{
		GaugeMetric: NewGaugeMetric("CPUutilization"),
	}
}

func (m *CPUutilization) Collect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, err := cpu.Percent(0, false)
	if err != nil {
		return
	}

	m.value = v[0]
}
