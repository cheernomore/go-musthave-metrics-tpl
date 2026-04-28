package repository

import models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"

type MetricsRepository interface {
	Save(metrics models.Metrics) error
	SaveBatch(metrics []models.Metrics) error
	Find(id string, metricType string) (models.Metrics, error)
	FindAll() ([]models.Metrics, error)
}

// FileStorage represents storage that can be persisted to a file
type FileStorage interface {
	SaveToFile(path string) error
	LoadFromFile(path string) error
}
