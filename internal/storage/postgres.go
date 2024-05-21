package storage

import (
	"context"
	"database/sql"
	"fmt"

	// Postgresql driver.
	_ "github.com/jackc/pgx/v5/stdlib"
)

var _ Storage = (*PostgresStorage)(nil)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return &PostgresStorage{
		db: db,
	}, nil
}

func (s *PostgresStorage) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("db.Close: %w", err)
	}

	return nil
}

func (s *PostgresStorage) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("db.PingContext: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetAllMetrics(ctx context.Context) map[string]Metric {
	_ = ctx

	return nil
}

func (s *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	_ = ctx
	_ = name

	return 0, nil
}

func (s *PostgresStorage) SetCounter(ctx context.Context, name string, value int64) error {
	_ = ctx
	_ = name
	_ = value

	return nil
}

func (s *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	_ = ctx
	_ = name

	return 0, nil
}

func (s *PostgresStorage) SetGauge(ctx context.Context, name string, value float64) error {
	_ = ctx
	_ = name
	_ = value

	return nil
}

func (s *PostgresStorage) LoadData(ctx context.Context, data map[string]Metric) error {
	_ = ctx
	_ = data

	return nil
}
