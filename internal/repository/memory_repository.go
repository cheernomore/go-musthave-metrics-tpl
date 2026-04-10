package repository

import (
	"encoding/json"
	"fmt"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"os"
	"sync"
)

type MemStorage struct {
	Metrics map[string]models.Metrics
	mu      sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Metrics: make(map[string]models.Metrics),
	}
}

func (m *MemStorage) Save(metric models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if metric.MType == models.Counter {
		current, ok := m.Metrics[metric.ID]
		if ok && current.Delta != nil {
			newVal := *current.Delta + *metric.Delta
			metric.Delta = &newVal
		}
	}

	m.Metrics[metric.ID] = metric
	return nil
}

func (m *MemStorage) SaveBatch(metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		if metric.MType == models.Counter {
			current, ok := m.Metrics[metric.ID]
			if ok && current.Delta != nil {
				newVal := *current.Delta + *metric.Delta
				metric.Delta = &newVal
			}
		}
		m.Metrics[metric.ID] = metric
	}
	return nil
}

func (m *MemStorage) Find(id string, metricType string) (models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.Metrics[id]
	if !ok {
		return models.Metrics{}, fmt.Errorf("metric %s not found", id)
	}
	if val.MType != metricType {
		return models.Metrics{}, fmt.Errorf("metric %s has type %s, not %s", id, val.MType, metricType)
	}
	return val, nil
}

func (m *MemStorage) FindAll() ([]models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	outputMetrics := make([]models.Metrics, 0, len(m.Metrics))
	for _, v := range m.Metrics {
		outputMetrics = append(outputMetrics, v)
	}
	return outputMetrics, nil
}

func (m *MemStorage) SaveToFile(path string) error {
	metrics, err := m.FindAll()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0666)
}

func (m *MemStorage) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, metric := range metrics {
		m.Save(metric)
	}
	return nil
}
