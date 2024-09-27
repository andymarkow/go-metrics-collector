// Package datamanager provides a metrics file writer.
package datamanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

// DataManager represents a data manager to load and save metrics data.
type DataManager struct {
	storeInterval time.Duration
	log           *zap.Logger
	storage       storage.Storage
	file          string
}

// NewDataManager creates a new DataManager instance.
//
// The storage parameter is required to store the metrics data and is used
// in the Load and Save methods.
func NewDataManager(storage storage.Storage, file string, opts ...Option) *DataManager {
	dm := &DataManager{
		log:           zap.NewNop(),
		file:          file,
		storage:       storage,
		storeInterval: 300 * time.Second,
	}

	// Apply options.
	for _, opt := range opts {
		opt(dm)
	}

	return dm
}

// Option represents a data manager option.
type Option func(d *DataManager)

// WithLogger sets the logger for the data manager.
func WithLogger(logger *zap.Logger) Option {
	return func(d *DataManager) {
		d.log = logger
	}
}

// WithStoreInterval sets the store interval for the data manager.
func WithStoreInterval(storeInterval time.Duration) Option {
	return func(d *DataManager) {
		d.storeInterval = storeInterval
	}
}

// Load loads the metrics data from the file.
func (m *DataManager) Load(ctx context.Context) error {
	m.log.Sugar().Infof("Loading data from file %s", m.file)

	data := make(map[string]storage.Metric)

	if err := readDataFromFile(m.file, &data); err != nil {
		return fmt.Errorf("failed to read data from file: %w", err)
	}

	if err := m.storage.LoadData(ctx, data); err != nil {
		return fmt.Errorf("storage.LoadData: %w", err)
	}

	return nil
}

func (m *DataManager) Save(ctx context.Context, file *os.File) error {
	data, err := m.storage.GetAllMetrics(ctx)
	if err != nil {
		return fmt.Errorf("storage.GetAllMetrics: %w", err)
	}

	if err := writeDataToFile(file, data); err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}

	return nil
}

func (m *DataManager) RunDataSaver(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	m.log.Info("Starting data saver")
	m.log.Sugar().Infof("Saving data every %s to the file %s", m.storeInterval.String(), m.file)

	f, err := os.OpenFile(m.file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}

	storeTicker := time.NewTicker(m.storeInterval)
	defer storeTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("Stopping data saver")
			m.log.Sugar().Infof("Flushing data to store file %s", m.file)

			if err := m.Save(ctx, f); err != nil {
				m.log.Error("failed to save data to store file", zap.Error(err))
			}

			if err := f.Close(); err != nil {
				return fmt.Errorf("file.Close: %w", err)
			}

			return nil

		case <-storeTicker.C:
			if err := m.Save(ctx, f); err != nil {
				m.log.Error("failed to save data to store file", zap.Error(err))
			}
		}
	}
}

func readDataFromFile(file string, data any) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			return
		}
	}()

	err = json.NewDecoder(f).Decode(&data)
	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	return nil
}

func writeDataToFile(file *os.File, data any) error {
	// Truncate the file content to 0.
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("file.Truncate: %w", err)
	}

	// Move the cursor to the beginning of the file.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("file.Seek: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")

	if err := encoder.Encode(&data); err != nil {
		return fmt.Errorf("encoder.Encode: %w", err)
	}

	// Sync the file content and write it to the disk.
	if err := file.Sync(); err != nil {
		return fmt.Errorf("file.Sync: %w", err)
	}

	return nil
}
