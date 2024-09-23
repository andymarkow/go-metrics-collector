// Package server provides a metrics server implementation.
package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/cryptutils"
	"github.com/andymarkow/go-metrics-collector/internal/datamanager"
	"github.com/andymarkow/go-metrics-collector/internal/logger"
	"github.com/andymarkow/go-metrics-collector/internal/server/httpserver"
	"github.com/andymarkow/go-metrics-collector/internal/server/httpserver/router"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

// Server represents a metrics server.
type Server struct {
	log           *zap.Logger
	httpsrv       *httpserver.HTTPServer
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

	privateKey, err := cryptutils.LoadRSAPrivateKey(cfg.CryptoKey)
	if err != nil {
		return nil, fmt.Errorf("cryptutils.LoadRSAPrivateKey: %w", err)
	}

	r := router.NewRouter(store,
		router.WithCryptoPrivateKey(privateKey),
		router.WithLogger(log),
		router.WithSignKey([]byte(cfg.SignKey)),
	)

	srv := httpserver.NewHTTPServer(r,
		httpserver.WithLogger(log),
		httpserver.WithServerAddr(cfg.ServerAddr),
	)

	datamgr := datamanager.NewDataManager(store, cfg.StoreFile,
		datamanager.WithLogger(log),
		datamanager.WithStoreInterval(time.Duration(cfg.StoreInterval)*time.Second),
	)

	return &Server{
		log:           log,
		httpsrv:       srv,
		datamgr:       datamgr,
		restoreOnBoot: cfg.RestoreOnBoot,
		storage:       store,
		storeInterval: time.Duration(cfg.StoreInterval) * time.Second,
		storeFile:     cfg.StoreFile,
	}, nil
}

// Close closes the server.
func (s *Server) Close() {
	if err := s.storage.Close(); err != nil {
		s.log.Error("storage.Close", zap.Error(err))
	}
}

// LoadDataFromFile loads the metrics data from the file.
func (s *Server) LoadDataFromFile(ctx context.Context) error {
	dataLoader, err := datamanager.NewDataLoader(s.storage, s.storeFile)
	if err != nil {
		return fmt.Errorf("datamanager.NewDataManager: %w", err)
	}

	s.log.Sugar().Infof("Loading data from file %s", s.storeFile)

	if err := dataLoader.Load(ctx); err != nil {
		return fmt.Errorf("dataLoader.Load: %w", err)
	}

	return nil
}

// SaveDataToFile saves the metrics data to the file.
func (s *Server) SaveDataToFile(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	dataSaver, err := datamanager.NewDataSaver(s.storage, s.storeFile)
	if err != nil {
		return fmt.Errorf("datamanager.NewDataSaver: %w", err)
	}
	defer func() {
		if err := dataSaver.Close(context.Background()); err != nil {
			s.log.Sugar().Errorf("dataSaver.Close: %v", err)
		}
	}()

	storeTicker := time.NewTicker(s.storeInterval)
	defer storeTicker.Stop()

	for {
		select {
		case <-ctx.Done():

			if err := dataSaver.PurgeAndSave(context.TODO()); err != nil { //nolint:contextcheck
				return fmt.Errorf("dataSaver.PurgeAndSave: %w", err)
			}

			return nil

		case <-storeTicker.C:
			if err := dataSaver.PurgeAndSave(context.TODO()); err != nil { //nolint:contextcheck
				s.log.Sugar().Errorf("dataSaver.PurgeAndSave: %v", err)
			}
		}
	}
}

// Start starts the server.
func (s *Server) Start() error {
	defer s.Close()

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
			errChan <- fmt.Errorf("server.Start: %w", err)
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
			s.log.Info("Gracefully shutting down server...")

			httpSrvStopCtx, httpSrvStopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer httpSrvStopCancel()

			if err := s.httpsrv.Shutdown(httpSrvStopCtx); err != nil {
				s.log.Error("server.Shutdown", zap.Error(err))
			}

			cancel()

			wg.Wait()

			return nil
		}
	}
}
