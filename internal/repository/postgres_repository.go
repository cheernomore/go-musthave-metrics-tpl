package repository

import (
	"context"
	"database/sql"
	"fmt"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"time"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (p *PostgresRepository) Save(metric models.Metrics) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if metric.MType == models.Counter {
		// Для counter используем UPSERT с INCREMENT
		query := `
			INSERT INTO metrics (name, type, delta, value)
			VALUES ($1, $2, $3, NULL)
			ON CONFLICT (name) DO UPDATE
			SET delta = metrics.delta + EXCLUDED.delta
		`
		_, err := p.db.ExecContext(ctx, query, metric.ID, metric.MType, metric.Delta)
		return err
	}

	// Для gauge используем простой UPSERT
	query := `
		INSERT INTO metrics (name, type, delta, value)
		VALUES ($1, $2, NULL, $3)
		ON CONFLICT (name) DO UPDATE
		SET value = EXCLUDED.value
	`
	_, err := p.db.ExecContext(ctx, query, metric.ID, metric.MType, metric.Value)
	return err
}

func (p *PostgresRepository) Find(id string, metricType string) (models.Metrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT name, type, delta, value FROM metrics WHERE name = $1 AND type = $2`

	var metric models.Metrics
	var delta sql.NullInt64
	var value sql.NullFloat64

	err := p.db.QueryRowContext(ctx, query, id, metricType).Scan(
		&metric.ID,
		&metric.MType,
		&delta,
		&value,
	)

	if err == sql.ErrNoRows {
		return models.Metrics{}, fmt.Errorf("metric %s not found", id)
	}
	if err != nil {
		return models.Metrics{}, err
	}

	if delta.Valid {
		metric.Delta = &delta.Int64
	}
	if value.Valid {
		metric.Value = &value.Float64
	}

	return metric, nil
}

func (p *PostgresRepository) FindAll() ([]models.Metrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT name, type, delta, value FROM metrics ORDER BY name`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.Metrics
	for rows.Next() {
		var metric models.Metrics
		var delta sql.NullInt64
		var value sql.NullFloat64

		if err := rows.Scan(&metric.ID, &metric.MType, &delta, &value); err != nil {
			return nil, err
		}

		if delta.Valid {
			metric.Delta = &delta.Int64
		}
		if value.Valid {
			metric.Value = &value.Float64
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return metrics, nil
}