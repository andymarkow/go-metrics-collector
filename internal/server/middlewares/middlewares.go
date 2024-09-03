// Package middlewares provides router middlewares.
package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/monitor"
)

// Middlewares is a collection of router middlewares.
type Middlewares struct {
	log     *zap.Logger
	signKey []byte
}

// New creates new Middlewares instance.
func New(opts ...Option) *Middlewares {
	// Default Middleware options.
	mw := &Middlewares{
		log: zap.Must(zap.NewDevelopment()),
	}

	// Apply options
	for _, opt := range opts {
		opt(mw)
	}

	return mw
}

// Option is a router middleware option.
type Option func(m *Middlewares)

// WithLogger is a router middleware option that sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(m *Middlewares) {
		m.log = logger
	}
}

// WithSignKey is a router middleware option that sets sign key.
func WithSignKey(signKey []byte) Option {
	return func(m *Middlewares) {
		m.signKey = signKey
	}
}

type responseData struct {
	status int
	size   int
}

// loggerResponseWriter wraps http.ResponseWriter and tracks the response size and status code.
// Uses in Logger middleware.
type loggerResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (w *loggerResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseData.size += size

	return size, err //nolint:wrapcheck
}

func (w *loggerResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.status = statusCode
}

// Logger is a router middleware that logs requests and their processing time.
func (m *Middlewares) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		responseData := &responseData{
			status: 200,
			size:   0,
		}

		cw := loggerResponseWriter{
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
