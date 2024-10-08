package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/monitor/metrics"
)

// MetricValidator is a router middleware that validates metric name and type.
func (m *Middlewares) MetricValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metricType")

		switch metricType {
		case string(metrics.MetricCounter), string(metrics.MetricGauge):
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
