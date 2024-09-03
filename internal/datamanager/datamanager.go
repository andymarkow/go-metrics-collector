// Package datamanager provides a metrics file writer.
package datamanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

// DataSaver is a metrics data saver to the file.
type DataSaver struct {
	file    *os.File
	encoder *json.Encoder
	storage storage.Storage
}

// NewDataSaver creates a new DataSaver instance.
//
// It opens the file with the specified name and creates a new json.Encoder
// from the file. The encoder is used to encode the metrics data into a JSON
// format. If the file doesn't exist, it will be created.
//
// The storage parameter is required to store the metrics data and is used
// in the Save and PurgeAndSave methods.
func NewDataSaver(fileName string, storage storage.Storage) (*DataSaver, error) {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile: %w", err)
	}

	return &DataSaver{
		file:    file,
		encoder: json.NewEncoder(file),
		storage: storage,
	}, nil
}

// Close closes the data saver.
func (d *DataSaver) Close() error {
	if err := d.file.Close(); err != nil {
		return fmt.Errorf("file.Close: %w", err)
	}

	return nil
}

// Save saves the metrics data to the file.
func (d *DataSaver) Save() error {
	ctx := context.TODO()

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
func (d *DataSaver) PurgeAndSave() error {
	if err := d.file.Truncate(0); err != nil {
		return fmt.Errorf("file.Truncate: %w", err)
	}

	if _, err := d.file.Seek(0, 0); err != nil {
		return fmt.Errorf("file.Seek: %w", err)
	}

	if err := d.Save(); err != nil {
		return fmt.Errorf("d.Save: %w", err)
	}

	return nil
}

// DataLoader is a metrics data loader from the file.
type DataLoader struct {
	file    *os.File
	decoder *json.Decoder
	storage storage.Storage
}

// NewDataLoader creates a new DataLoader instance.
//
// It opens the file with the specified name and creates a new json.Decoder
// from the file. The decoder is used to decode the file content into a
// Metrics struct. If the file doesn't exist, it will be created.
//
// The storage parameter is required to store the metrics data and is used
// in the Load method.
func NewDataLoader(fileName string, storage storage.Storage) (*DataLoader, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile: %w", err)
	}

	return &DataLoader{
		file:    file,
		decoder: json.NewDecoder(file),
		storage: storage,
	}, nil
}

// Close closes the data loader.
func (d *DataLoader) Close() error {
	if err := d.file.Close(); err != nil {
		return fmt.Errorf("file.Close: %w", err)
	}

	return nil
}

// GetFilename returns the name of the file.
func (d *DataLoader) GetFilename() string {
	return d.file.Name()
}

// Load loads the metrics data from the file.
func (d *DataLoader) Load() error {
	data := make(map[string]storage.Metric)

	err := d.decoder.Decode(&data)
	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	ctx := context.Background()

	if err := d.storage.LoadData(ctx, data); err != nil {
		return fmt.Errorf("storage.LoadData: %w", err)
	}

	return nil
}
