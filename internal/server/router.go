package server

import (
	"github.com/andymarkow/go-metrics-collector/internal/handlers"
	"github.com/andymarkow/go-metrics-collector/internal/server/middlewares"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type chiRouter struct {
	chi.Router
	log     *zap.Logger
	signKey []byte
}

type Option func(r *chiRouter)

func newRouter(strg storage.Storage, opts ...Option) chiRouter {
	r := chiRouter{
		Router: chi.NewRouter(),
		log:    zap.NewNop(),
	}

	for _, opt := range opts {
		opt(&r)
	}

	h := handlers.NewHandlers(strg, handlers.WithLogger(r.log))

	mw := middlewares.New(
		middlewares.WithLogger(r.log),
		middlewares.WithSignKey(r.signKey),
	)

	r.Use(
		middleware.Recoverer,
		middleware.StripSlashes,
		mw.Logger,
		mw.Compress,
	)

	var useHashSumValidator bool

	if len(r.signKey) > 0 {
		useHashSumValidator = true
	}

	r.Get("/", h.GetAllMetrics)
	r.Get("/ping", h.Ping)

	r.Group(func(r chi.Router) {
		r.Use(mw.MetricValidator)
		r.Get("/value/{metricType}/{metricName}", h.GetMetric)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetric)
	})

	r.Group(func(r chi.Router) {
		r.Post("/value", h.GetMetricJSON)
		r.Post("/update", h.UpdateMetricJSON)
	})

	r.Group(func(r chi.Router) {
		if useHashSumValidator {
			r.Use(mw.MetricValidator)
		}

		r.Post("/updates", h.UpdateMetricsJSON)
	})

	return r
}

func WithLogger(logger *zap.Logger) Option {
	return func(r *chiRouter) {
		r.log = logger
	}
}

func WithSignKey(signKey []byte) Option {
	return func(r *chiRouter) {
		r.signKey = signKey
	}
}
