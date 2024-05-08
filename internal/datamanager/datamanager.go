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

type DataSaver struct {
	file    *os.File
	encoder *json.Encoder
	storage storage.Storage
}

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

func (d *DataSaver) Close() error {
	if err := d.file.Close(); err != nil {
		return fmt.Errorf("file.Close: %w", err)
	}

	return nil
}

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

type DataLoader struct {
	file    *os.File
	decoder *json.Decoder
	storage storage.Storage
}

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

func (d *DataLoader) Close() error {
	if err := d.file.Close(); err != nil {
		return fmt.Errorf("file.Close: %w", err)
	}

	return nil
}

func (d *DataLoader) GetFilename() string {
	return d.file.Name()
}

func (d *DataLoader) Load() error {
	data := make(map[string]storage.Metric)

	err := d.decoder.Decode(&data)
	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	ctx := context.TODO()

	if err := d.storage.LoadData(ctx, data); err != nil {
		return fmt.Errorf("storage.LoadData: %w", err)
	}

	return nil
}
