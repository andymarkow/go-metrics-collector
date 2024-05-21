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

type Server struct {
	srv           *http.Server
	log           *zap.Logger
	storage       storage.Storage
	storeInterval time.Duration
	storeFile     string
	restoreOnBoot bool
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

	var strg storage.Storage = storage.NewMemStorage()

	if cfg.DatabaseDSN != "" {
		pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("storage.NewPostgresStorage: %w", err)
		}

		ctx := context.TODO()

		if err := pgStorage.Bootstrap(ctx); err != nil {
			return nil, fmt.Errorf("pgStorage.Bootstrap: %w", err)
		}

		strg = pgStorage
	}

	store := storage.NewStorage(strg)

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
		storeFile:     cfg.StoreFile,
	}, nil
}

func (s *Server) Close() {
	if err := s.storage.Close(); err != nil {
		s.log.Error("storage.Close", zap.Error(err))
	}
}

func (s *Server) LoadDataFromFile() error {
	dataLoader, err := datamanager.NewDataLoader(s.storeFile, s.storage)
	if err != nil {
		return fmt.Errorf("datamanager.NewDataManager: %w", err)
	}
	defer dataLoader.Close()

	s.log.Sugar().Infof("Loading metrics data from file '%s'", dataLoader.GetFilename())

	if err := dataLoader.Load(); err != nil {
		return fmt.Errorf("dataLoader.Load: %w", err)
	}

	return nil
}

func (s *Server) SaveDataToFile(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	dataSaver, err := datamanager.NewDataSaver(s.storeFile, s.storage)
	if err != nil {
		return fmt.Errorf("datamanager.NewDataSaver: %w", err)
	}
	defer dataSaver.Close()

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

	// Graceful shutdown handler
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
