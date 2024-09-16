// Package server provides a metrics server implementation.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/datamanager"
	"github.com/andymarkow/go-metrics-collector/internal/logger"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

// Server represents a metrics server.
type Server struct {
	log           *zap.Logger
	srv           *http.Server
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

	r := newRouter(store, WithLogger(log), WithSignKey([]byte(cfg.SignKey)))

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	return &Server{
		srv:           srv,
		log:           log,
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
func (s *Server) LoadDataFromFile() error {
	dataLoader, err := datamanager.NewDataLoader(s.storeFile, s.storage)
	if err != nil {
		return fmt.Errorf("datamanager.NewDataManager: %w", err)
	}
	defer func() {
		if err := dataLoader.Close(); err != nil {
			s.log.Sugar().Errorf("dataLoader.Close: %v", err)
		}
	}()

	s.log.Sugar().Infof("Loading data from file '%s'", dataLoader.GetFilename())

	if err := dataLoader.Load(); err != nil {
		return fmt.Errorf("dataLoader.Load: %w", err)
	}

	return nil
}

// SaveDataToFile saves the metrics data to the file.
func (s *Server) SaveDataToFile(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	dataSaver, err := datamanager.NewDataSaver(s.storeFile, s.storage)
	if err != nil {
		return fmt.Errorf("datamanager.NewDataSaver: %w", err)
	}
	defer func() {
		if err := dataSaver.Close(); err != nil {
			s.log.Sugar().Errorf("dataSaver.Close: %v", err)
		}
	}()

	storeTicker := time.NewTicker(s.storeInterval)
	defer storeTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			if err := dataSaver.PurgeAndSave(); err != nil { //nolint:contextcheck
				return fmt.Errorf("dataSaver.PurgeAndSave: %w", err)
			}

			return nil

		case <-storeTicker.C:
			if err := dataSaver.PurgeAndSave(); err != nil { //nolint:contextcheck
				s.log.Sugar().Errorf("dataSaver.PurgeAndSave: %v", err)
			}
		}
	}
}

// Start starts the server.
func (s *Server) Start() error {
	defer s.Close()

	if s.restoreOnBoot {
		if err := s.LoadDataFromFile(); err != nil {
			return fmt.Errorf("server.LoadData: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)

	wg := &sync.WaitGroup{}

	if s.storeFile != "" {
		wg.Add(1)

		s.log.Sugar().Infof("Saving data to file '%s' every %s", s.storeFile, s.storeInterval.String())

		go func() {
			if err := s.SaveDataToFile(ctx, wg); err != nil {
				errChan <- fmt.Errorf("server.SaveData: %w", err)
			}
		}()
	}

	go func() {
		s.log.Sugar().Infof("Starting server on '%s'", s.srv.Addr)

		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server.ListenAndServe: %w", err)
		}
	}()

	// Graceful shutdown handler.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			return err

		case <-quit:
			s.log.Sugar().Infof("Gracefully shutting down server...")

			cancel()

			wg.Wait()

			return nil
		}
	}
}
