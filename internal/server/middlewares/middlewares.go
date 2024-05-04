package middlewares

import (
	"net/http"
	"time"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/monitor"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Middlewares struct {
	log *zap.Logger
}

type Config struct {
	Logger *zap.Logger
}

type responseData struct {
	status int
	size   int
}

type customResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (w *customResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseData.size += size

	return size, err //nolint:wrapcheck
}

func (w *customResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.status = statusCode
}

func New(cfg *Config) *Middlewares {
	return &Middlewares{
		log: cfg.Logger,
	}
}

// MetricValidator is a router middleware that validates metric name and type.
func (m *Middlewares) MetricValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metricType")

		switch metricType {
		case string(monitor.MetricCounter), string(monitor.MetricGauge):
		default:
			http.Error(w, errormsg.ErrMetricInvalidType.Error(), http.StatusBadRequest)

			return
		}

		metricName := chi.URLParam(r, "metricName")
		if metricName == "" {
			http.Error(w, errormsg.ErrMetricEmptyName.Error(), http.StatusNotFound)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middlewares) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		responseData := &responseData{
			status: 200,
			size:   0,
		}

		cw := customResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		defer func() {
			m.log.Info("request",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Int("status", responseData.status),
				zap.Int("size", responseData.size),
				zap.Duration("duration_ms", time.Since(startTime)*1000),
			)
		}()

		next.ServeHTTP(&cw, r)
	})
}
