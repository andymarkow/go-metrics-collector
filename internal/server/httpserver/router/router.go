// Package router provides HTTP server router.
package router

import (
	"crypto/rsa"
	"net"
	_ "net/http/pprof" //nolint:gosec // Enable pprof debugger

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/server/httpserver/router/handlers"
	"github.com/andymarkow/go-metrics-collector/internal/server/httpserver/router/middlewares"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

type routerOpts struct {
	logger        *zap.Logger
	trustedSubnet *net.IPNet
	cryptoPrivKey *rsa.PrivateKey
	signKey       []byte
}

func NewRouter(store storage.Storage, opts ...Option) *chi.Mux {
	rOpts := &routerOpts{
		logger:  zap.NewNop(),
		signKey: make([]byte, 0),
	}

	for _, opt := range opts {
		opt(rOpts)
	}

	h := handlers.NewHandlers(store, handlers.WithLogger(rOpts.logger))

	r := chi.NewRouter()

	mw := middlewares.New(
		middlewares.WithLogger(rOpts.logger),
		middlewares.WithSignKey(rOpts.signKey),
		middlewares.WithCryptoPrivateKey(rOpts.cryptoPrivKey),
		middlewares.WithTrustedSubnet(rOpts.trustedSubnet),
	)

	r.Use(
		middleware.Recoverer,
		middleware.StripSlashes,
		mw.Logger,
		mw.Whitelist,
	)

	var useHashSumValidator bool

	if len(rOpts.signKey) > 0 {
		useHashSumValidator = true
	}

	r.Mount("/debug", middleware.Profiler())

	r.Get("/ping", h.Ping)
	r.With(mw.Compress).Get("/", h.GetAllMetrics)

	r.Group(func(r chi.Router) {
		r.Use(mw.Compress)
		r.Use(mw.MetricValidator)

		r.Get("/value/{metricType}/{metricName}", h.GetMetric)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetric)
	})

	r.Group(func(r chi.Router) {
		r.Use(mw.Compress)

		r.Post("/value", h.GetMetricJSON)
		r.Post("/update", h.UpdateMetricJSON)
	})

	r.Group(func(r chi.Router) {
		r.Use(mw.Compress)

		if rOpts.cryptoPrivKey != nil {
			r.Use(mw.Cryptography)
		}

		if useHashSumValidator {
			r.Use(mw.HashSumValidator)
		}

		r.Post("/updates", h.UpdateMetricsJSON)
	})

	return r
}

// Option is a router option.
type Option func(o *routerOpts)

// WithLogger is a router option that sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(o *routerOpts) {
		o.logger = logger
	}
}

// WithSignKey is a router option that sets sign key.
func WithSignKey(signKey []byte) Option {
	return func(o *routerOpts) {
		o.signKey = signKey
	}
}

// WithCryptoPrivateKey is a router option that sets decription RSA private key.
func WithCryptoPrivateKey(key *rsa.PrivateKey) Option {
	return func(o *routerOpts) {
		o.cryptoPrivKey = key
	}
}

// WithTrustedSubnet is a router option that sets trusted subnet.
func WithTrustedSubnet(subnet *net.IPNet) Option {
	return func(o *routerOpts) {
		o.trustedSubnet = subnet
	}
}
