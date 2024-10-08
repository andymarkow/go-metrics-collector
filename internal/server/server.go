// Package server provides a metrics server implementation.
package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/reflection"

	"github.com/andymarkow/go-metrics-collector/internal/datamanager"
	metricpbv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/api/metric/v1"
	grpcserver "github.com/andymarkow/go-metrics-collector/internal/grpc/server"
	metricsvcv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/service/metric/v1"
	"github.com/andymarkow/go-metrics-collector/internal/logger"
	"github.com/andymarkow/go-metrics-collector/internal/server/httpserver"
	"github.com/andymarkow/go-metrics-collector/internal/server/httpserver/router"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
	"github.com/andymarkow/go-metrics-collector/internal/tlsutils"
)

// Server represents a metrics server.
type Server struct {
	log           *zap.Logger
	httpsrv       *httpserver.HTTPServer
	grpcsrv       *grpcserver.Server
	datamgr       *datamanager.DataManager
	storage       storage.Storage
	storeFile     string
	storeInterval time.Duration
	restoreOnBoot bool
}

// NewServer creates a new metrics server.
func NewServer() (*Server, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("newConfig: %w", err)
	}

	log, err := logger.NewZapLogger(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("logger.NewZapLogger: %w", err)
	}

	var strg storage.Storage = storage.NewMemStorage()

	if cfg.DatabaseDSN != "" {
		pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseDSN, storage.WithLogger(log))
		if err != nil {
			return nil, fmt.Errorf("storage.NewPostgresStorage: %w", err)
		}

		ctx := context.Background()

		if err := pgStorage.Bootstrap(ctx); err != nil {
			return nil, fmt.Errorf("pgStorage.Bootstrap: %w", err)
		}

		strg = pgStorage
	}

	store := storage.NewStorage(strg)

	var privateKey *rsa.PrivateKey

	if cfg.CryptoKey != "" {
		log.Info("Loading crypto key " + cfg.CryptoKey)

		privateKey, err = tlsutils.LoadRSAPrivateKey(cfg.CryptoKey)
		if err != nil {
			return nil, fmt.Errorf("tlsutils.LoadRSAPrivateKey: %w", err)
		}
	}

	var subnet *net.IPNet

	if cfg.TrustedSubnet != "" {
		var err error
		_, subnet, err = net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trusted subnet: %w", err)
		}
	}

	r := router.NewRouter(store,
		router.WithLogger(log),
		router.WithSignKey([]byte(cfg.SignKey)),
		router.WithCryptoPrivateKey(privateKey),
		router.WithTrustedSubnet(subnet),
	)

	httpsrv := httpserver.NewHTTPServer(r,
		httpserver.WithLogger(log),
		httpserver.WithServerAddr(cfg.ServerAddr),
	)

	datamgr := datamanager.NewDataManager(store, cfg.StoreFile,
		datamanager.WithLogger(log),
		datamanager.WithStoreInterval(time.Duration(cfg.StoreInterval)*time.Second),
	)

	msvcv1 := metricsvcv1.NewMetricService(store,
		metricsvcv1.WithLogger(log),
		metricsvcv1.WithSignKey([]byte(cfg.SignKey)),
	)

	grpcsrv := grpcserver.NewServer(
		grpcserver.WithLogger(log),
		grpcserver.WithServerAddr(cfg.GrpcServerAddr),
	)

	metricpbv1.RegisterMetricServiceServer(grpcsrv.Server(), msvcv1)

	reflection.Register(grpcsrv.Server())

	return &Server{
		log:           log,
		httpsrv:       httpsrv,
		grpcsrv:       grpcsrv,
		datamgr:       datamgr,
		restoreOnBoot: cfg.RestoreOnBoot,
		storage:       store,
		storeInterval: time.Duration(cfg.StoreInterval) * time.Second,
		storeFile:     cfg.StoreFile,
	}, nil
}

// Close closes the server.
func (s *Server) Close() error {
	if err := s.storage.Close(); err != nil {
		return fmt.Errorf("storage.Close: %w", err)
	}

	return nil
}

// Start starts the server.
func (s *Server) Start() error {
	defer func() {
		if err := s.Close(); err != nil {
			s.log.Error("failed to close server", zap.Error(err))

			return
		}
	}()

	if s.restoreOnBoot {
		if err := s.datamgr.Load(context.Background()); err != nil {
			return fmt.Errorf("failed to load data: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)

	wg := &sync.WaitGroup{}

	if s.storeFile != "" {
		wg.Add(1)

		go func() {
			if err := s.datamgr.RunDataSaver(ctx, wg); err != nil {
				errChan <- fmt.Errorf("datamanager.RunDataSaver: %w", err)
			}
		}()
	}

	go func() {
		if err := s.httpsrv.Start(); err != nil {
			errChan <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	go func() {
		if err := s.grpcsrv.Serve(); err != nil {
			errChan <- fmt.Errorf("failed to start gRPC server: %w", err)
		}
	}()

	// Graceful shutdown handler.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case err := <-errChan:
			return err

		case <-quit:
			s.log.Info("Gracefully shutting down server")

			httpSrvStopCtx, httpSrvStopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer httpSrvStopCancel()

			if err := s.httpsrv.Shutdown(httpSrvStopCtx); err != nil {
				s.log.Error("server.Shutdown", zap.Error(err))
			}

			grpcSrvStopCtx, grpcSrvStopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer grpcSrvStopCancel()

			if err := s.grpcsrv.Shutdown(grpcSrvStopCtx); err != nil {
				s.log.Error("server.Shutdown", zap.Error(err))
			}

			cancel()

			wg.Wait()

			return nil
		}
	}
}
