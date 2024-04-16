package monitor

import (
	"runtime"
	"testing"
)

func TestCollect(t *testing.T) {
	// Prepare a fake MemStats object for testing
	fakeMemStats := &runtime.MemStats{
		Alloc: 1024, // Set a sample value for Alloc
	}

	// Create a new Alloc instance
	metric := newAllocMetric(fakeMemStats)

	// Call Collect to collect the metric value
	metric.Collect()

	// Check if the collected value is correct
	expectedValue := float64(1024)
	if metric.value != expectedValue {
		t.Errorf("Expected collected value to be %f, got %f", expectedValue, metric.value)
	}
}
