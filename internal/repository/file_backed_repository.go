package repository

import (
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
)

// FileBackedRepository is a decorator that automatically saves to file after each Save operation
type FileBackedRepository struct {
	repo     MetricsRepository
	filePath string
}

// NewFileBackedRepository creates a repository that automatically persists changes to a file
func NewFileBackedRepository(repo MetricsRepository, filePath string) *FileBackedRepository {
	return &FileBackedRepository{
		repo:     repo,
		filePath: filePath,
	}
}

func (f *FileBackedRepository) Save(metric models.Metrics) error {
	// Save to underlying repository
	err := f.repo.Save(metric)
	if err != nil {
		return err
	}

	// Synchronously save to file if the underlying repo supports it
	if fileStorage, ok := f.repo.(FileStorage); ok {
		if saveErr := fileStorage.SaveToFile(f.filePath); saveErr != nil {
			// Log error but don't fail the operation
			// The metric is already saved in memory
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