package repository

import (
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
)

// FileBackedRepository — декоратор над MetricsRepository, который синхронно
// сбрасывает состояние в файл после каждой операции записи (если вложенное
// хранилище реализует FileStorage).
type FileBackedRepository struct {
	repo     MetricsRepository
	filePath string
}

// NewFileBackedRepository оборачивает repo, добавляя синхронное сохранение
// в файл filePath после каждой записи.
func NewFileBackedRepository(repo MetricsRepository, filePath string) *FileBackedRepository {
	return &FileBackedRepository{
		repo:     repo,
		filePath: filePath,
	}
}

// Save сохраняет метрику во вложенное хранилище и сбрасывает состояние в файл.
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

// SaveBatch сохраняет набор метрик и сбрасывает состояние в файл.
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

// Find делегирует поиск метрики вложенному хранилищу.
func (f *FileBackedRepository) Find(id string, metricType string) (models.Metrics, error) {
	return f.repo.Find(id, metricType)
}

// FindAll делегирует выборку всех метрик вложенному хранилищу.
func (f *FileBackedRepository) FindAll() ([]models.Metrics, error) {
	return f.repo.FindAll()
}
