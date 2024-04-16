package server

import (
	"fmt"
	"log"
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

func newRouter(strg storage.Storage) chi.Router {
	h := handlers.NewHandlers(strg)

	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		middleware.StripSlashes,
	)

	r.Get("/", h.GetAllMetrics)

	r.Group(func(r chi.Router) {
		r.Use(metricValidatorMW)
		r.Get("/value/{metricType}/{metricName}", h.GetMetric)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetric)
	})

	return r
}

func NewServer() (*Server, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("newConfig: %w", err)
	}

	strg := storage.NewStorage(storage.NewMemStorage())

	r := newRouter(strg)

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	return &Server{
		srv: srv,
	}, nil
}

func (s *Server) Start() error {
	log.Printf("Starting server on %q\n", s.srv.Addr)

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server.ListenAndServe: %w", err)
	}

	return nil
}

// metricValidatorMW is a router middleware that validates metric name and type.
func metricValidatorMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metricType")

		switch metricType {
		case string(monitor.MetricCounter), string(monitor.MetricGauge):
		default:
			http.Error(w, handlers.ErrMetricInvalidType.Error(), http.StatusBadRequest)

			return
		}

		metricName := chi.URLParam(r, "metricName")
		if metricName == "" {
			http.Error(w, handlers.ErrMetricEmptyName.Error(), http.StatusNotFound)

			return
		}

		next.ServeHTTP(w, r)
	})
}
