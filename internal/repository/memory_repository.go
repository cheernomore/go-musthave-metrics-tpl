package repository

import (
	"fmt"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
)

type MemStorage struct {
	Metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Metrics: make(map[string]models.Metrics),
	}
}

func (m *MemStorage) Save(metric models.Metrics) error {
	switch metric.MType {
	case models.Counter:
		if existing, ok := m.Metrics[metric.ID]; ok {
			newDelta := *existing.Delta + *metric.Delta
			metric.Delta = &newDelta
		}
		m.Metrics[metric.ID] = metric
	case models.Gauge:
		m.Metrics[metric.ID] = metric
	}
	return nil
}

func (m *MemStorage) Find(id string, metricType string) (models.Metrics, error) {
	if val, ok := m.Metrics[id]; ok {
		if val.MType == metricType {
			return val, nil
		}
		return models.Metrics{}, fmt.Errorf("metric %s has type %s, not %s", id, val.MType, metricType)
	}
	return models.Metrics{}, fmt.Errorf("metric %s not found", id)
}

func (m *MemStorage) FindAll() ([]models.Metrics, error) {
	var outputMetrics []models.Metrics
	for _, v := range m.Metrics {
		outputMetrics = append(outputMetrics, v)
	}
	return outputMetrics, nil
}
