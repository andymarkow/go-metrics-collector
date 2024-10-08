// Package httpserver provides a HTTP server implementation.
package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type HTTPServer struct {
	log    *zap.Logger
	server *http.Server
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(router http.Handler, opts ...Option) *HTTPServer {
	srv := &HTTPServer{
		server: &http.Server{
			Addr:              ":8080",
			Handler:           router,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      60 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

// Option is a HTTP server option.
type Option func(s *HTTPServer)

// WithServerAddr is a HTTP server option that sets server address.
func WithServerAddr(addr string) Option {
	return func(s *HTTPServer) {
		s.server.Addr = addr
	}
}

// WithReadTimeout is a HTTP server option that sets server read timeout.
func WithReadTimeout(timeout time.Duration) Option {
	return func(s *HTTPServer) {
		s.server.ReadTimeout = timeout
	}
}

// WithReadHeaderTimeout is a HTTP server option that sets server read header timeout.
func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *HTTPServer) {
		s.server.ReadHeaderTimeout = timeout
	}
}

// WithWriteTimeout is a HTTP server option that sets server write timeout.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(s *HTTPServer) {
		s.server.WriteTimeout = timeout
	}
}

// WithLogger is a HTTP server option that sets logger.
func WithLogger(log *zap.Logger) Option {
	return func(s *HTTPServer) {
		s.log = log
	}
}

// Start starts the HTTP server.
func (s *HTTPServer) Start() error {
	s.log.Info("Starting HTTP server", zap.String("address", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server.ListenAndServe: %w", err)
	}

	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down HTTP server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server.Shutdown: %w", err)
	}

	return nil
}
