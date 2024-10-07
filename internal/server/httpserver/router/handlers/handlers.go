// Package handlers provides HTTP handlers.
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

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

// Handlers is a collection of router handlers.
type Handlers struct {
	log     *zap.Logger
	storage storage.Storage
}

// NewHandlers returns a new Handlers instance.
func NewHandlers(strg storage.Storage, opts ...Option) *Handlers {
	handlers := &Handlers{
		storage: strg,
		log:     zap.NewNop(),
	}

	// Apply options
	for _, opt := range opts {
		opt(handlers)
	}

	return handlers
}

// Option is a functional option type for Handlers.
type Option func(h *Handlers)

// WithLogger is an option for Handlers instance that sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(h *Handlers) {
		h.log = logger
	}
}

// Ping handles ping request.
func (h *Handlers) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.storage.Ping(r.Context()); err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	h.checkRespError(w.Write([]byte("OK")))
}

// GetAllMetrics handles get all metrics request.
func (h *Handlers) GetAllMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	result := make([]string, 0)

	data, err := h.storage.GetAllMetrics(ctx)
	if err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	for k, v := range data {
		result = append(result, fmt.Sprintf("%s %s", k, v.StringValue()))
	}

	slices.Sort(result)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	h.checkRespError(w.Write([]byte(strings.Join(result, "\n"))))
}

func (h *Handlers) GetMetric(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metricName := chi.URLParam(r, "metricName")
	metricType := chi.URLParam(r, "metricType")

	var metricValue string

	switch metricType {
	case string(metrics.MetricCounter):
		val, err := h.storage.GetCounter(ctx, metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			h.handleError(w, err, http.StatusNotFound)

			return
		} else if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricValue = fmt.Sprintf("%d", val)

	case string(metrics.MetricGauge):
		val, err := h.storage.GetGauge(ctx, metricName)
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

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	h.checkRespError(io.WriteString(w, metricValue))
}

func (h *Handlers) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
	case string(metrics.MetricCounter):
		if err := h.storage.SetCounter(ctx, metricName, int64(metricValue)); err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}
	case string(metrics.MetricGauge):
		if err := h.storage.SetGauge(ctx, metricName, metricValue); err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}
	default:
		h.handleError(w, errormsg.ErrMetricInvalidType, http.StatusBadRequest)

		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	h.checkRespError(w.Write([]byte(http.StatusText(http.StatusOK))))
}

func (h *Handlers) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
	case string(metrics.MetricCounter):
		val, err := h.storage.GetCounter(ctx, metricPayload.ID)
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

	case string(metrics.MetricGauge):
		val, err := h.storage.GetGauge(ctx, metricPayload.ID)
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
	}

	resp, err := json.Marshal(metricResult)
	if err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	h.checkRespError(w.Write(resp))
}

func (h *Handlers) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var metricPayload models.Metrics
	var metricResult models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metricPayload); err != nil {
		if errors.Is(err, io.EOF) {
			h.handleError(w, errormsg.ErrEmptyRequestPayload, http.StatusBadRequest)

			return
		}

		h.handleError(w, err, http.StatusBadRequest)

		return
	}

	h.log.Sugar().Debugf("payload: %+v", metricPayload)

	if err := metricPayload.ValidateUpdate(); err != nil {
		h.handleError(w, err, http.StatusBadRequest)

		return
	}

	switch metricPayload.MType {
	case string(metrics.MetricCounter):
		if err := h.storage.SetCounter(ctx, metricPayload.ID, *metricPayload.Delta); err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		val, err := h.storage.GetCounter(ctx, metricPayload.ID)
		if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricResult = models.Metrics{
			ID:    metricPayload.ID,
			MType: metricPayload.MType,
			Delta: &val,
		}

	case string(metrics.MetricGauge):
		if err := h.storage.SetGauge(ctx, metricPayload.ID, *metricPayload.Value); err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricResult = models.Metrics{
			ID:    metricPayload.ID,
			MType: metricPayload.MType,
			Value: metricPayload.Value,
		}
	}

	resp, err := json.Marshal(metricResult)
	if err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	h.checkRespError(w.Write(resp))
}

func (h *Handlers) UpdateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var metricsPayload []models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metricsPayload); err != nil {
		if errors.Is(err, io.EOF) {
			h.handleError(w, errormsg.ErrEmptyRequestPayload, http.StatusBadRequest)

			return
		}

		h.handleError(w, err, http.StatusBadRequest)

		return
	}

	h.log.Sugar().Debugf("payload: %+v", metricsPayload)

	for _, metric := range metricsPayload {
		if err := metric.ValidateUpdate(); err != nil {
			h.handleError(w, err, http.StatusBadRequest)

			return
		}
	}

	if err := h.storage.SetMetrics(ctx, metricsPayload); err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	h.checkRespError(w.Write([]byte("OK")))
}

// parseGaugeMetricValue parses gauge metric value from string.
func parseGaugeMetricValue(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat: %w", err)
	}

	return v, nil
}

func (h *Handlers) checkRespError(_ int, err error) {
	if err != nil {
		h.log.Error("failed to write response", zap.Error(err))
	}
}

// handleError handles error response.
func (h *Handlers) handleError(
	w http.ResponseWriter, err error, statusCode int,
) {
	h.log.Error(err.Error())
	http.Error(w, err.Error(), statusCode)
}
