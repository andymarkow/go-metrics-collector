package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

type Handlers struct {
	mStorage *storage.MemStorage
}

func NewHandlers(mStorage *storage.MemStorage) *Handlers {
	return &Handlers{mStorage: mStorage}
}

func (h *Handlers) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricName := r.PathValue("metricName")
	if metricName == "" {
		http.Error(w, "empty metric name", http.StatusNotFound)
		return
	}

	metricValueRaw := r.PathValue("metricValue")
	if metricValueRaw == "" {
		http.Error(w, "empty metric value", http.StatusBadRequest)
		return
	}

	metricValue, err := parseMetricValue(metricValueRaw)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid metric value (%s): %v", metricValueRaw, err.Error()), http.StatusBadRequest)
		return
	}

	switch r.PathValue("metricType") {
	case "counter":
		h.mStorage.SetCounter(metricName, int64(metricValue))
	case "gauge":
		h.mStorage.SetGauge(metricName, metricValue)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func parseMetricValue(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat: %w", err)
	}

	return v, nil
}
