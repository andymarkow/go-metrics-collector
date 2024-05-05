package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/datamanager"
	"github.com/andymarkow/go-metrics-collector/internal/logger"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

type Server struct {
	srv           *http.Server
	log           *zap.Logger
	storage       storage.Storage
	storeInterval time.Duration
	restoreOnBoot bool
	dataLoader    *datamanager.DataLoader
	dataSaver     *datamanager.DataSaver
}

func NewServer() (*Server, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, fmt.Errorf("newConfig: %w", err)
	}

	log, err := logger.NewZapLogger(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("logger.NewZapLogger: %w", err)
	}

	pgstore, err := storage.NewPostgresStorage(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("storage.NewPostgresStorage: %w", err)
	}

	store := storage.NewStorage(pgstore)

	dl, err := datamanager.NewDataLoader(cfg.StoreFile, store)
	if err != nil {
		return nil, fmt.Errorf("datamanager.NewDataManager: %w", err)
	}

	ds, err := datamanager.NewDataSaver(cfg.StoreFile, store)
	if err != nil {
		return nil, fmt.Errorf("datamanager.NewDataSaver: %w", err)
	}

	r := newRouter(store, WithLogger(log))

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	return &Server{
		srv:           srv,
		log:           log,
		restoreOnBoot: cfg.RestoreOnBoot,
		storage:       store,
		storeInterval: time.Duration(cfg.StoreInterval) * time.Second,
		dataLoader:    dl,
		dataSaver:     ds,
	}, nil
}

func (s *Server) Close() {
	if err := s.dataSaver.Close(); err != nil {
		s.log.Error("dataSaver.Close", zap.Error(err))
	}

	if err := s.storage.Close(); err != nil {
		s.log.Error("storage.Close", zap.Error(err))
	}
}

func (s *Server) LoadData() error {
	defer s.dataLoader.Close()
	if s.restoreOnBoot {
		s.log.Sugar().Infof("Loading metrics data from file '%s'", s.dataLoader.GetFilename())

		if err := s.dataLoader.Load(); err != nil {
			return fmt.Errorf("dataLoader.Load: %w", err)
		}
	}

	return nil
}

func (s *Server) Start() error {
	defer s.Close()

	if err := s.LoadData(); err != nil {
		return fmt.Errorf("server.LoadData: %w", err)
	}

	s.log.Sugar().Infof("Starting server on '%s'", s.srv.Addr)

	errChan := make(chan error, 1)

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server.ListenAndServe: %w", err)
		}
	}()

	storeTicker := time.NewTicker(s.storeInterval)

	defer storeTicker.Stop()

	// Graceful shutdown handler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			return err

		case <-quit:
			s.log.Sugar().Infof("Gracefully shutting down server...")

			if err := s.dataSaver.PurgeAndSave(); err != nil {
				return fmt.Errorf("dataSaver.PurgeAndSave: %w", err)
			}

			return nil

		case <-storeTicker.C:
			if err := s.dataSaver.PurgeAndSave(); err != nil {
				s.log.Sugar().Errorf("dataSaver.PurgeAndSave: %v", err)
			}
		}
	}
}
