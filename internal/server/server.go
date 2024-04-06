package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"

	"github.com/andymarkow/go-metrics-collector/internal/handlers"
	"github.com/andymarkow/go-metrics-collector/internal/monitor"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

type Server struct {
	srv *http.Server
}

func NewServer() *Server {
	memStorage := storage.NewStorage(storage.NewMemStorage())

	h := handlers.NewHandlers(memStorage)

	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
	)

	r.Get("/", h.GetAllMetrics)

	r.Group(func(r chi.Router) {
		r.Use(metricValidatorMW)
		r.Get("/value/{metricType}/{metricName}", h.GetMetric)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetric)
	})

	srv := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           r,
		ReadTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	return &Server{
		srv: srv,
	}
}

func (s *Server) Start() error {
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server.ListenAndServe: %w", err)
	}

	return nil
}

func metricValidatorMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metricType")

		switch metricType {
		case string(monitor.MetricCounter), string(monitor.MetricGauge):
		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)

			return
		}

		metricName := chi.URLParam(r, "metricName")
		if metricName == "" {
			http.Error(w, "empty metric name", http.StatusNotFound)

			return
		}

		next.ServeHTTP(w, r)
	})
}
