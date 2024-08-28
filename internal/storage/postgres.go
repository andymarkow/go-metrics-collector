package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/andymarkow/go-metrics-collector/internal/models"
	"github.com/pressly/goose/v3"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

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

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(180 * time.Second)
	db.SetConnMaxLifetime(3600 * time.Second)

	return &PostgresStorage{
		db: db,
	}, nil
}

func (pg *PostgresStorage) Bootstrap(ctx context.Context) error {
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		pg.db,
		os.DirFS("migrations"),
	)
	if err != nil {
		return fmt.Errorf("goose.NewProvider: %w", err)
	}

	_, err = provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("provider.Up: %w", err)
	}

	return nil
}

func (pg *PostgresStorage) Close() error {
	if err := pg.db.Close(); err != nil {
		return fmt.Errorf("db.Close: %w", err)
	}

	return nil
}

func (pg *PostgresStorage) Ping(ctx context.Context) error {
	err := WithRetry(func() error {
		if err := pg.db.PingContext(ctx); err != nil {
			return fmt.Errorf("db.PingContext: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (pg *PostgresStorage) GetAllMetrics(ctx context.Context) (map[string]Metric, error) {
	data := make(map[string]Metric)

	err := WithRetry(func() error {
		countersStmt, err := pg.db.PrepareContext(ctx, "SELECT name, value FROM metric_counters;")
		if err != nil {
			return fmt.Errorf("db.PrepareContext: %w", err)
		}
		defer countersStmt.Close()

		counters, err := countersStmt.QueryContext(ctx)
		if err != nil {
			return fmt.Errorf("countersStmt.QueryContext: %w", err)
		}
		defer counters.Close()

		for counters.Next() {
			var name string
			var value int64

			if err := counters.Scan(&name, &value); err != nil {
				return fmt.Errorf("counters.Scan: %w", err)
			}

			data[name] = Metric{
				Type:  "counter",
				Value: value,
			}
		}

		if err := counters.Err(); err != nil {
			return fmt.Errorf("counters.Err: %w", err)
		}

		gaugesStmt, err := pg.db.PrepareContext(ctx, "SELECT name, value FROM metric_gauges;")
		if err != nil {
			return fmt.Errorf("db.PrepareContext: %w", err)
		}
		defer gaugesStmt.Close()

		gauges, err := gaugesStmt.QueryContext(ctx)
		if err != nil {
			return fmt.Errorf("gaugesStmt.QueryContext: %w", err)
		}
		defer gauges.Close()

		for gauges.Next() {
			var name string
			var value float64

			if err := gauges.Scan(&name, &value); err != nil {
				return fmt.Errorf("gauges.Scan: %w", err)
			}

			data[name] = Metric{
				Type:  "gauge",
				Value: value,
			}
		}

		if err := gauges.Err(); err != nil {
			return fmt.Errorf("gauges.Err: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (pg *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	var value int64

	err := WithRetry(func() error {
		stmt, err := pg.db.PrepareContext(ctx, "SELECT value FROM metric_counters WHERE name = $1;")
		if err != nil {
			return fmt.Errorf("db.PrepareContext: %w", err)
		}
		defer stmt.Close()

		row := stmt.QueryRowContext(ctx, name)

		err = row.Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMetricNotFound
		} else if err != nil {
			return fmt.Errorf("row.Scan: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return value, nil
}

func (pg *PostgresStorage) SetCounter(ctx context.Context, name string, value int64) error {
	query := `
		INSERT INTO metric_counters (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name)
		DO UPDATE SET value = metric_counters.value + $2;`

	err := WithRetry(func() error {
		stmt, err := pg.db.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("db.PrepareContext: %w", err)
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx, name, value)
		if err != nil {
			return fmt.Errorf("stmt.ExecContext: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (pg *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	var value float64

	err := WithRetry(func() error {
		stmt, err := pg.db.PrepareContext(ctx, "SELECT value FROM metric_gauges WHERE name = $1;")
		if err != nil {
			return fmt.Errorf("db.PrepareContext: %w", err)
		}
		defer stmt.Close()

		row := stmt.QueryRowContext(ctx, name)

		if err := row.Scan(&value); errors.Is(err, sql.ErrNoRows) {
			return ErrMetricNotFound
		} else if err != nil {
			return fmt.Errorf("row.Scan: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return value, nil
}

func (pg *PostgresStorage) SetGauge(ctx context.Context, name string, value float64) error {
	query := `
		INSERT INTO metric_gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name)
		DO UPDATE SET value = $2;`

	err := WithRetry(func() error {
		stmt, err := pg.db.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("db.PrepareContext: %w", err)
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx, name, value)
		if err != nil {
			return fmt.Errorf("stmt.ExecContext: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (pg *PostgresStorage) SetMetrics(ctx context.Context, metrics []models.Metrics) error {
	err := WithRetry(func() error {
		tx, err := pg.db.Begin()
		if err != nil {
			return fmt.Errorf("db.Begin: %w", err)
		}
		defer tx.Rollback() //nolint:errcheck

		counterStmt, err := tx.PrepareContext(ctx,
			"INSERT INTO metric_counters (name, value) VALUES ($1, $2)"+
				"ON CONFLICT (name) DO UPDATE SET value = metric_counters.value + $2;")
		if err != nil {
			return fmt.Errorf("tx.PrepareContext: %w", err)
		}
		defer counterStmt.Close()

		gaugeStmt, err := tx.PrepareContext(ctx,
			"INSERT INTO metric_gauges (name, value) VALUES ($1, $2)"+
				"ON CONFLICT (name) DO UPDATE SET value = $2;")
		if err != nil {
			return fmt.Errorf("tx.PrepareContext: %w", err)
		}
		defer gaugeStmt.Close()

		for _, metric := range metrics {
			switch metric.MType {
			case "counter":
				_, err := counterStmt.ExecContext(ctx, metric.ID, *metric.Delta)
				if err != nil {
					return fmt.Errorf("counterStmt.ExecContext: %w", err)
				}

			case "gauge":
				_, err := gaugeStmt.ExecContext(ctx, metric.ID, *metric.Value)
				if err != nil {
					return fmt.Errorf("gaugeStmt.ExecContext: %w", err)
				}

			default:
				return fmt.Errorf("unknown metric type: %s", metric.MType)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("tx.Commit: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// LoadData is a stub to keep compatibility with Storage interface.
func (pg *PostgresStorage) LoadData(_ context.Context, _ map[string]Metric) error {
	return nil
}

// WithRetry retries operations in case of retryable errors.
func WithRetry(operation func() error) error {
	// Retry count
	retryCount := 3

	// Initial retry wait time
	var retryWaitTime time.Duration

	// Define the interval between retries
	retryWaitInterval := 2

	var err error

	for i := range retryCount {
		err = operation()
		if err == nil {
			return nil
		}

		if isRetryableError(err) {
			retryWaitTime = time.Duration((i*retryWaitInterval + 1)) * time.Second // 1s, 3s, 5s, etc.

			// TODO: time.After or time.Ticker.
			time.Sleep(retryWaitTime)
		} else {
			return fmt.Errorf("%w", err)
		}
	}

	return fmt.Errorf("retry attempts exceeded: %w", err)
}

// isRetryableError checks if error is retryable.
func isRetryableError(err error) bool {
	// Connection refused error
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code) {
		// https://github.com/jackc/pgerrcode/blob/6e2875d9b438d43808cc033afe2d978db3b9c9e7/errcode.go#L393C6-L393C27
		return true
	}

	return false
}
