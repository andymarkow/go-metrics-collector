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
	log           *zap.Logger
	file          string
	storage       storage.Storage
	storeInterval time.Duration
}

// NewDataManager creates a new DataManager instance.
//
// The storage parameter is required to store the metrics data and is used
// in the Load and Save methods.
func NewDataManager(storage storage.Storage, file string, opts ...DataManagerOpt) *DataManager {
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

// DataManagerOpt represents a data manager option.
type DataManagerOpt func(d *DataManager)

// WithLogger sets the logger for the data manager.
func WithLogger(log *zap.Logger) DataManagerOpt {
	return func(d *DataManager) {
		d.log = log
	}
}

// WithStoreInterval sets the store interval for the data manager.
func WithStoreInterval(storeInterval time.Duration) DataManagerOpt {
	return func(d *DataManager) {
		d.storeInterval = storeInterval
	}
}

// Load loads the metrics data from the file.
func (m *DataManager) Load(ctx context.Context) error {
	f, err := os.OpenFile(m.file, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			m.log.Error("failed to close the file", zap.Error(err))

			return
		}
	}()

	data := make(map[string]storage.Metric)

	err = json.NewDecoder(f).Decode(&data)
	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
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

// DataSaver is a metrics data saver to the file.
type DataSaver struct {
	log           *zap.Logger
	file          *os.File
	encoder       *json.Encoder
	storage       storage.Storage
	storeInterval time.Duration
}

// NewDataSaver creates a new DataSaver instance.
//
// It opens the file with the specified name and creates a new json.Encoder
// from the file. The encoder is used to encode the metrics data into a JSON
// format. If the file doesn't exist, it will be created.
//
// The storage parameter is required to store the metrics data and is used
// in the Save and PurgeAndSave methods.
func NewDataSaver(storage storage.Storage, file string, opts ...DataSaverOpt) (*DataSaver, error) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile: %w", err)
	}

	ds := &DataSaver{
		log:           zap.NewNop(),
		file:          f,
		encoder:       json.NewEncoder(f),
		storage:       storage,
		storeInterval: 300 * time.Second,
	}

	for _, opt := range opts {
		opt(ds)
	}

	return ds, nil
}

// DataSaverOpt is a DataSaver option.
type DataSaverOpt func(d *DataSaver)

// Run starts the data saver. It periodically saves the data to the file
// and to the storage. When the context is canceled, it saves the data one
// more time and returns.
//
// The data saver is stopped if the context is canceled. The function returns
// an error if the data saver fails to save the data to the storage.
func (d *DataSaver) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	storeTicker := time.NewTicker(d.storeInterval)
	defer storeTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-storeTicker.C:
			if err := d.PurgeAndSave(ctx); err != nil {
				d.log.Error("failed to save data to store file", zap.Error(err))
			}
		}
	}
}

// Close closes the data saver.
func (d *DataSaver) Close(ctx context.Context) error {
	if err := d.PurgeAndSave(ctx); err != nil {
		return fmt.Errorf("dataSaver.Save: %w", err)
	}

	if err := d.file.Close(); err != nil {
		return fmt.Errorf("file.Close: %w", err)
	}

	return nil
}

// Save saves the metrics data to the file.
func (d *DataSaver) Save(ctx context.Context) error {
	data, err := d.storage.GetAllMetrics(ctx)
	if err != nil {
		return fmt.Errorf("storage.GetAllMetrics: %w", err)
	}

	d.encoder.SetIndent("", "\t")

	if err := d.encoder.Encode(&data); err != nil {
		return fmt.Errorf("encoder.Encode: %w", err)
	}

	return nil
}

// PurgeAndSave purges the file and saves the metrics data.
func (d *DataSaver) PurgeAndSave(ctx context.Context) error {
	if err := d.file.Truncate(0); err != nil {
		return fmt.Errorf("file.Truncate: %w", err)
	}

	if _, err := d.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("file.Seek: %w", err)
	}

	if err := d.Save(ctx); err != nil {
		return fmt.Errorf("datasaver.Save: %w", err)
	}

	return nil
}

// DataLoader is a metrics data loader from the file.
type DataLoader struct {
	file    string
	storage storage.Storage
}

// NewDataLoader creates a new DataLoader instance.
func NewDataLoader(storage storage.Storage, file string) (*DataLoader, error) {
	return &DataLoader{
		file:    file,
		storage: storage,
	}, nil
}

// Load loads the metrics data from the file.
func (d *DataLoader) Load(ctx context.Context) error {
	f, err := os.OpenFile(d.file, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			return
		}
	}()

	data := make(map[string]storage.Metric)

	decoder := json.NewDecoder(f)

	err = decoder.Decode(&data)
	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	if err := d.storage.LoadData(ctx, data); err != nil {
		return fmt.Errorf("storage.LoadData: %w", err)
	}

	return nil
}
