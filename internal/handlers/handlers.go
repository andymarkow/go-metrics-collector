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

// Handlers is a collection of handlers.
type Handlers struct {
	storage storage.Storage
	log     *zap.Logger
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

// Option is a functional option for Handlers.
type Option func(h *Handlers)

// WithLogger is a option for Handlers that sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(h *Handlers) {
		h.log = logger
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

func (h *Handlers) Ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.storage.Ping(ctx); err != nil {
		h.handleError(w, err, http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

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

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strings.Join(result, "\n")))
}

func (h *Handlers) GetMetric(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metricName := chi.URLParam(r, "metricName")
	metricType := chi.URLParam(r, "metricType")

	var metricValue string

	switch metricType {
	case string(monitor.MetricCounter):
		val, err := h.storage.GetCounter(ctx, metricName)
		if errors.Is(err, storage.ErrMetricNotFound) {
			h.handleError(w, err, http.StatusNotFound)

			return
		} else if err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}

		metricValue = fmt.Sprintf("%d", val)

	case string(monitor.MetricGauge):
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

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, metricValue)
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
	case string(monitor.MetricCounter):
		if err := h.storage.SetCounter(ctx, metricName, int64(metricValue)); err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}
	case string(monitor.MetricGauge):
		if err := h.storage.SetGauge(ctx, metricName, metricValue); err != nil {
			h.handleError(w, err, http.StatusInternalServerError)

			return
		}
	default:
		h.handleError(w, errormsg.ErrMetricInvalidType, http.StatusBadRequest)

		return
	}

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
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
	case string(monitor.MetricCounter):
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

	case string(monitor.MetricGauge):
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

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
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
	case string(monitor.MetricCounter):
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

	case string(monitor.MetricGauge):
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

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
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

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
