//nolint:errcheck
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/monitor"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handlers struct {
	storage storage.Storage
	log     *zap.Logger
}

func NewHandlers(strg storage.Storage, log *zap.Logger) *Handlers {
	return &Handlers{
		storage: strg,
		log:     log,
	}
}

func parseGaugeMetricValue(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat: %w", err)
	}

	return v, nil
}

func (h *Handlers) handleError(
	w http.ResponseWriter, err error, statusCode int,
) {
	h.log.Error(err.Error())
	http.Error(w, err.Error(), statusCode)
}

func (h *Handlers) GetAllMetrics(w http.ResponseWriter, _ *http.Request) {
	result := make([]string, 0)

	for k, v := range h.storage.GetAllMetrics() {
		result = append(result, fmt.Sprintf("%s %s", k, v))
	}

	slices.Sort(result)

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strings.Join(result, "\n")))
}

func (h *Handlers) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		h.handleError(w, errormsg.ErrMetricEmptyName, http.StatusNotFound)

		return
	}

	metricType := chi.URLParam(r, "metricType")

	var metricValue string

	switch metricType {
	case string(monitor.MetricCounter):
		val, err := h.storage.GetCounter(metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			h.handleError(w, err, http.StatusNotFound)

			return
		} else if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricValue = fmt.Sprintf("%d", val)

	case string(monitor.MetricGauge):
		val, err := h.storage.GetGauge(metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			h.handleError(w, err, http.StatusNotFound)

			return
		} else if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		// Remove trailing zeros in string value to make check tests pass
		// More info: https://github.com/andymarkow/go-metrics-collector/actions/runs/8584210095/job/23524237884#step:11:32
		metricValue = strconv.FormatFloat(val, 'f', -1, 64)

	default:
		h.handleError(w, errormsg.ErrMetricInvalidType, http.StatusBadRequest)

		return
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, metricValue)
}

func (h *Handlers) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metricName")

	metricValueRaw := chi.URLParam(r, "metricValue")
	if metricValueRaw == "" {
		h.handleError(w, errormsg.ErrMetricEmptyValue, http.StatusBadRequest)

		return
	}

	metricValue, err := parseGaugeMetricValue(metricValueRaw)
	if err != nil {
		h.handleError(w, errormsg.ErrMetricInvalidValue, http.StatusBadRequest)

		return
	}

	metricType := chi.URLParam(r, "metricType")

	switch metricType {
	case string(monitor.MetricCounter):
		h.storage.SetCounter(metricName, int64(metricValue))
	case string(monitor.MetricGauge):
		h.storage.SetGauge(metricName, metricValue)
	default:
		h.handleError(w, errormsg.ErrMetricInvalidType, http.StatusBadRequest)

		return
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (h *Handlers) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metricPayload models.Metrics
	var metricResult models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metricPayload); err != nil {
		if errors.Is(err, io.EOF) {
			h.handleError(w, errormsg.ErrEmptyRequestPayload, http.StatusBadRequest)

			return
		}

		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	h.log.Sugar().Debugf("payload: %+v", metricPayload)

	if err := metricPayload.Validate(); err != nil {
		h.handleError(w, err, http.StatusBadRequest)

		return
	}

	switch metricPayload.MType {
	case string(monitor.MetricCounter):
		if metricPayload.Delta == nil {
			h.handleError(w, errormsg.ErrMetricEmptyDelta, http.StatusBadRequest)

			return
		}

		h.storage.SetCounter(metricPayload.ID, int64(*metricPayload.Delta))

		val, err := h.storage.GetCounter(metricPayload.ID)
		if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricResult = models.Metrics{
			ID:    metricPayload.ID,
			MType: metricPayload.MType,
			Delta: &val,
		}

	case string(monitor.MetricGauge):
		if metricPayload.Value == nil {
			h.handleError(w, errormsg.ErrMetricEmptyValue, http.StatusBadRequest)

			return
		}

		h.storage.SetGauge(metricPayload.ID, *metricPayload.Value)

		metricResult = models.Metrics{
			ID:    metricPayload.ID,
			MType: metricPayload.MType,
			Value: metricPayload.Value,
		}

	default:
		h.handleError(w, errormsg.ErrMetricInvalidType, http.StatusBadRequest)

		return
	}

	resp, err := json.Marshal(metricResult)
	if err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handlers) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metricPayload models.Metrics
	var metricResult models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metricPayload); err != nil {
		if errors.Is(err, io.EOF) {
			h.handleError(w, errormsg.ErrEmptyRequestPayload, http.StatusBadRequest)

			return
		}

		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	if err := metricPayload.Validate(); err != nil {
		h.handleError(w, err, http.StatusBadRequest)

		return
	}

	switch metricPayload.MType {
	case string(monitor.MetricCounter):
		val, err := h.storage.GetCounter(metricPayload.ID)
		if errors.Is(err, storage.ErrMetricNotFound) {
			h.handleError(w, err, http.StatusNotFound)

			return
		} else if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricResult = models.Metrics{
			ID:    metricPayload.ID,
			MType: metricPayload.MType,
			Delta: &val,
		}

	case string(monitor.MetricGauge):
		val, err := h.storage.GetGauge(metricPayload.ID)
		if errors.Is(err, storage.ErrMetricNotFound) {
			h.handleError(w, err, http.StatusNotFound)

			return
		} else if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricResult = models.Metrics{
			ID:    metricPayload.ID,
			MType: metricPayload.MType,
			Value: &val,
		}

	default:
		h.handleError(w, errormsg.ErrMetricInvalidType, http.StatusBadRequest)

		return
	}

	resp, err := json.Marshal(metricResult)
	if err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
