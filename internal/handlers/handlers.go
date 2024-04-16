package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

type Handlers struct {
	memStorage *storage.MemStorage
}

func NewHandlers(memStorage *storage.MemStorage) *Handlers {
	return &Handlers{memStorage: memStorage}
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
		http.Error(w, fmt.Sprintf("invalid metric value (%q): %v", metricValueRaw, err.Error()), http.StatusBadRequest)

		return
	}

	switch r.PathValue("metricType") {
	case "counter":
		h.memStorage.SetCounter(metricName, int64(metricValue))
	case "gauge":
		h.memStorage.SetGauge(metricName, metricValue)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)

		return
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK))) //nolint:errcheck
}

func parseMetricValue(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat: %w", err)
	}

	return v, nil
}
