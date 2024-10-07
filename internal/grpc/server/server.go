// Package server provides a gRPC server implementation.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	srv *grpc.Server
	cfg *config
	log *zap.Logger
}

type config struct {
	addr    string
	network string
}

func NewServer(opts ...Option) *Server {
	srv := &Server{
		srv: grpc.NewServer(),
		cfg: defaultConfig(),
		log: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

func defaultConfig() *config {
	return &config{
		addr:    ":8080",
		network: "tcp",
	}
}

type Option func(c *Server)

func WithServerAddr(addr string) Option {
	return func(c *Server) {
		c.cfg.addr = addr
	}
}

func WithServerNetwork(network string) Option {
	return func(c *Server) {
		c.cfg.network = network
	}
}

func WithLogger(log *zap.Logger) Option {
	return func(c *Server) {
		c.log = log
	}
}

func (s *Server) Server() *grpc.Server {
	return s.srv
}

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
