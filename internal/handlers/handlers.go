//nolint:errcheck
package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/andymarkow/go-metrics-collector/internal/monitor"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	storage storage.Storage
}

func NewHandlers(strg storage.Storage) *Handlers {
	return &Handlers{storage: strg}
}

func (h *Handlers) GetAllMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)

	for k, v := range h.storage.GetAllMetrics() {
		fmt.Fprintln(w, k, v)
	}
}

func (h *Handlers) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "empty metric name", http.StatusNotFound)

		return
	}

	metricType := chi.URLParam(r, "metricType")

	var metricValue string

	switch metricType {
	case string(monitor.MetricCounter):
		val, err := h.storage.GetCounter(metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		metricValue = fmt.Sprintf("%d", val)

	case string(monitor.MetricGauge):
		val, err := h.storage.GetGauge(metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Remove trailing zeros in string value to make check tests pass
		// More info: https://github.com/andymarkow/go-metrics-collector/actions/runs/8584210095/job/23524237884#step:11:32
		metricValue = strconv.FormatFloat(val, 'f', -1, 64)
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, metricValue)
}

func (h *Handlers) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")

	metricValueRaw := chi.URLParam(r, "metricValue")
	if metricValueRaw == "" {
		http.Error(w, "empty metric value", http.StatusBadRequest)

		return
	}

	metricValue, err := parseGaugeMetricValue(metricValueRaw)
	if err != nil {
		http.Error(w,
			fmt.Sprintf("invalid metric value (%q): %v", metricValueRaw, err.Error()),
			http.StatusBadRequest,
		)

		return
	}

	metricType := chi.URLParam(r, "metricType")

	switch metricType {
	case string(monitor.MetricCounter):
		h.storage.SetCounter(metricName, int64(metricValue))
	case string(monitor.MetricGauge):
		h.storage.SetGauge(metricName, metricValue)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)

		return
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func parseGaugeMetricValue(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat: %w", err)
	}

	return v, nil
}
