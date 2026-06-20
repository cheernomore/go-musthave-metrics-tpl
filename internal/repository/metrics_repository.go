// Package repository предоставляет реализации хранилища метрик:
// in-memory, файловое и PostgreSQL.
package repository

import models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"

// MetricsRepository — абстракция хранилища метрик. Реализуется MemStorage,
// FileBackedRepository и PostgresRepository.
type MetricsRepository interface {
	// Save сохраняет одну метрику. Для counter значение прибавляется к текущему.
	Save(metrics models.Metrics) error
	// SaveBatch сохраняет набор метрик за одну операцию.
	SaveBatch(metrics []models.Metrics) error
	// Find возвращает метрику по имени и типу или ошибку, если она не найдена.
	Find(id string, metricType string) (models.Metrics, error)
	// FindAll возвращает все сохранённые метрики.
	FindAll() ([]models.Metrics, error)
}

// FileStorage реализуется хранилищами, умеющими сохранять своё состояние
// в файл и восстанавливать его из файла.
type FileStorage interface {
	// SaveToFile сериализует метрики в файл по указанному пути.
	SaveToFile(path string) error
	// LoadFromFile загружает метрики из файла по указанному пути.
	LoadFromFile(path string) error
}
