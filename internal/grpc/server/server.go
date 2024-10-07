// Package server provides a gRPC server implementation.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	_ "google.golang.org/grpc/encoding/gzip"

	"github.com/andymarkow/go-metrics-collector/internal/grpc/interceptor"
)

// Server represents a gRPC server.
type Server struct {
	srv *grpc.Server
	cfg *config
	log *zap.Logger
}

// config represents a gRPC server configuration.
type config struct {
	addr    string
	network string
}

// NewServer creates a new gRPC server.
func NewServer(opts ...Option) *Server {
	srv := &Server{
		cfg: defaultConfig(),
		log: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(srv)
	}

	logInterc := interceptor.NewLogInterceptor(interceptor.WithLogInterceptorLogger(srv.log))

	grpcsrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logInterc.ServerLogInterceptor),
	)

	srv.srv = grpcsrv

	return srv
}

// defaultConfig returns the default config for the server.
func defaultConfig() *config {
	return &config{
		addr:    ":8080",
		network: "tcp",
	}
}

// Option represents a gRPC server option.
type Option func(c *Server)

// WithServerAddr sets the gRPC server address.
func WithServerAddr(addr string) Option {
	return func(c *Server) {
		c.cfg.addr = addr
	}
}

// WithServerNetwork sets the gRPC server network.
func WithServerNetwork(network string) Option {
	return func(c *Server) {
		c.cfg.network = network
	}
}

// WithLogger sets the logger for the gRPC server.
func WithLogger(log *zap.Logger) Option {
	return func(c *Server) {
		c.log = log
	}
}

// Server returns the gRPC server.
func (s *Server) Server() *grpc.Server {
	return s.srv
}

// Serve starts the gRPC server.
func (s *Server) Serve() error {
	listener, err := net.Listen(s.cfg.network, s.cfg.addr)
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	s.log.Info("Starting gRPC server", zap.String("address", s.cfg.addr))
	if err := s.srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("srv.Serve: %w", err)
	}

	return nil
}

// Shutdown shuts down the gRPC server.
func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	defer close(done)

	go func() {
		s.log.Info("Gracefully shutting down gRPC server")

		s.srv.GracefulStop()

		done <- struct{}{}
	}()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Forcibly shutting down gRPC server")

			s.srv.Stop()

			s.log.Info("gRPC server stopped")

			return nil

		case <-done:
			s.log.Info("gRPC server stopped")

			return nil
		}
	}
}
