package repository

import (
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
)

type FileBackedRepository struct {
	repo     MetricsRepository
	filePath string
}

func NewFileBackedRepository(repo MetricsRepository, filePath string) *FileBackedRepository {
	return &FileBackedRepository{
		repo:     repo,
		filePath: filePath,
	}
}

func (f *FileBackedRepository) Save(metric models.Metrics) error {
	err := f.repo.Save(metric)
	if err != nil {
		return err
	}

	if fileStorage, ok := f.repo.(FileStorage); ok {
		if saveErr := fileStorage.SaveToFile(f.filePath); saveErr != nil {
			return saveErr
		}
	}

	return nil
}

func (f *FileBackedRepository) SaveBatch(metrics []models.Metrics) error {
	err := f.repo.SaveBatch(metrics)
	if err != nil {
		return err
	}

	if fileStorage, ok := f.repo.(FileStorage); ok {
		if saveErr := fileStorage.SaveToFile(f.filePath); saveErr != nil {
			return saveErr
		}
	}

	return nil
}

func (f *FileBackedRepository) Find(id string, metricType string) (models.Metrics, error) {
	return f.repo.Find(id, metricType)
}

func (f *FileBackedRepository) FindAll() ([]models.Metrics, error) {
	return f.repo.FindAll()
}
