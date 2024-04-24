package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/handlers"
	"github.com/andymarkow/go-metrics-collector/internal/logger"
	"github.com/andymarkow/go-metrics-collector/internal/server/middlewares"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

type Server struct {
	srv *http.Server
	log *zap.Logger
}

type routerConfig struct {
	storage storage.Storage
	logger  *zap.Logger
}

func newRouter(cfg *routerConfig) chi.Router {
	h := handlers.NewHandlers(cfg.storage, cfg.logger)

	mw := middlewares.New(&middlewares.Config{
		Logger: cfg.logger,
	})

	r := chi.NewRouter()
	r.Use(
		middleware.Recoverer,
		middleware.StripSlashes,
		mw.Logger,
	)

	r.Get("/", h.GetAllMetrics)

	r.Group(func(r chi.Router) {
		r.Use(mw.MetricValidator)
		r.Get("/value/{metricType}/{metricName}", h.GetMetric)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetric)
	})

	r.Group(func(r chi.Router) {
		r.Post("/update", h.UpdateMetricJSON)
		r.Post("/value", h.GetMetricJSON)
	})

	return r
}

func NewServer() (*Server, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("newConfig: %w", err)
	}

	log, err := logger.NewZapLogger(&logger.Config{
		Level: cfg.LogLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("logger.NewZapLogger: %w", err)
	}

	strg := storage.NewStorage(storage.NewMemStorage())

	r := newRouter(&routerConfig{
		storage: strg,
		logger:  log,
	})

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	return &Server{
		srv: srv,
		log: log,
	}, nil
}

func (s *Server) Start() error {
	s.log.Sugar().Infof("Starting server on '%s'", s.srv.Addr)

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server.ListenAndServe: %w", err)
	}

	return nil
}
