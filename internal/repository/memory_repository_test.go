package repository

import (
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMemStorage_Save(t *testing.T) {
	tests := []struct {
		name    string
		metrics []models.Metrics
		want    map[string]models.Metrics
	}{
		{
			name: "save single gauge",
			metrics: []models.Metrics{
				{
					ID:    "TestGauge",
					MType: "gauge",
					Value: func() *float64 { v := 123.45; return &v }(),
				},
			},
			want: map[string]models.Metrics{
				"TestGauge": {
					ID:    "TestGauge",
					MType: "gauge",
					Value: func() *float64 { v := 123.45; return &v }(),
				},
			},
		},
		{
			name: "save counter - accumulate",
			metrics: []models.Metrics{
				{
					ID:    "TestCounter",
					MType: "counter",
					Delta: func() *int64 { v := int64(10); return &v }(),
				},
				{
					ID:    "TestCounter",
					MType: "counter",
					Delta: func() *int64 { v := int64(5); return &v }(),
				},
			},
			want: map[string]models.Metrics{
				"TestCounter": {
					ID:    "TestCounter",
					MType: "counter",
					Delta: func() *int64 { v := int64(15); return &v }(),
				},
			},
		},
		{
			name: "update gauge - overwrite",
			metrics: []models.Metrics{
				{
					ID:    "TestGauge",
					MType: "gauge",
					Value: func() *float64 { v := 100.0; return &v }(),
				},
				{
					ID:    "TestGauge",
					MType: "gauge",
					Value: func() *float64 { v := 200.0; return &v }(),
				},
			},
			want: map[string]models.Metrics{
				"TestGauge": {
					ID:    "TestGauge",
					MType: "gauge",
					Value: func() *float64 { v := 200.0; return &v }(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMemStorage()
			for _, m := range tt.metrics {
				err := repo.Save(m)
				require.NoError(t, err)
			}

			for key, expectedMetric := range tt.want {
				actualMetric, ok := repo.Metrics[key]
				require.True(t, ok, "Metric %s not found", key)
				assert.Equal(t, expectedMetric.ID, actualMetric.ID)
				assert.Equal(t, expectedMetric.MType, actualMetric.MType)
				if expectedMetric.Value != nil {
					require.NotNil(t, actualMetric.Value)
					assert.Equal(t, *expectedMetric.Value, *actualMetric.Value)
				}
				if expectedMetric.Delta != nil {
					require.NotNil(t, actualMetric.Delta)
					assert.Equal(t, *expectedMetric.Delta, *actualMetric.Delta)
				}
			}
		})
	}
}

func TestMemStorage_Find(t *testing.T) {
	repo := NewMemStorage()
	gaugeValue := 123.45
	counterDelta := int64(10)

	_ = repo.Save(models.Metrics{
		ID:    "TestGauge",
		MType: "gauge",
		Value: &gaugeValue,
	})
	_ = repo.Save(models.Metrics{
		ID:    "TestCounter",
		MType: "counter",
		Delta: &counterDelta,
	})

	tests := []struct {
		name       string
		id         string
		metricType string
		wantErr    bool
		wantValue  *float64
		wantDelta  *int64
	}{
		{
			name:       "find existing gauge",
			id:         "TestGauge",
			metricType: "gauge",
			wantErr:    false,
			wantValue:  &gaugeValue,
		},
		{
			name:       "find existing counter",
			id:         "TestCounter",
			metricType: "counter",
			wantErr:    false,
			wantDelta:  &counterDelta,
		},
		{
			name:       "find non-existing metric",
			id:         "NonExistent",
			metricType: "gauge",
			wantErr:    true,
		},
		{
			name:       "find with wrong type",
			id:         "TestGauge",
			metricType: "counter",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := repo.Find(tt.id, tt.metricType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.id, metric.ID)
				assert.Equal(t, tt.metricType, metric.MType)
				if tt.wantValue != nil {
					require.NotNil(t, metric.Value)
					assert.Equal(t, *tt.wantValue, *metric.Value)
				}
				if tt.wantDelta != nil {
					require.NotNil(t, metric.Delta)
					assert.Equal(t, *tt.wantDelta, *metric.Delta)
				}
			}
		})
	}
}

func TestMemStorage_FindAll(t *testing.T) {
	repo := NewMemStorage()
	gaugeValue := 123.45
	counterDelta := int64(10)

	_ = repo.Save(models.Metrics{
		ID:    "TestGauge",
		MType: "gauge",
		Value: &gaugeValue,
	})
	_ = repo.Save(models.Metrics{
		ID:    "TestCounter",
		MType: "counter",
		Delta: &counterDelta,
	})

	metrics, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, metrics, 2)

	// Проверяем, что оба значения есть в результате
	found := make(map[string]bool)
	for _, m := range metrics {
		found[m.ID] = true
	}
	assert.True(t, found["TestGauge"])
	assert.True(t, found["TestCounter"])
}

func TestMemStorage_SaveToFileAndLoadFromFile(t *testing.T) {
	// Создаем временный файл
	tmpFile, err := os.CreateTemp("", "metrics_test_*.json")
	require.NoError(t, err)
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Создаем репозиторий и добавляем метрики
	repo1 := NewMemStorage()
	gaugeValue := 123.45
	counterDelta := int64(10)

	_ = repo1.Save(models.Metrics{
		ID:    "TestGauge",
		MType: "gauge",
		Value: &gaugeValue,
	})
	_ = repo1.Save(models.Metrics{
		ID:    "TestCounter",
		MType: "counter",
		Delta: &counterDelta,
	})

	// Сохраняем в файл
	err = repo1.SaveToFile(tmpPath)
	require.NoError(t, err)

	// Загружаем из файла в новый репозиторий
	repo2 := NewMemStorage()
	err = repo2.LoadFromFile(tmpPath)
	require.NoError(t, err)

	// Проверяем, что метрики загружены корректно
	gauge, err := repo2.Find("TestGauge", "gauge")
	require.NoError(t, err)
	assert.Equal(t, "TestGauge", gauge.ID)
	assert.Equal(t, "gauge", gauge.MType)
	require.NotNil(t, gauge.Value)
	assert.Equal(t, gaugeValue, *gauge.Value)

	counter, err := repo2.Find("TestCounter", "counter")
	require.NoError(t, err)
	assert.Equal(t, "TestCounter", counter.ID)
	assert.Equal(t, "counter", counter.MType)
	require.NotNil(t, counter.Delta)
	assert.Equal(t, counterDelta, *counter.Delta)
}

func TestMemStorage_LoadFromFile_FileNotExists(t *testing.T) {
	repo := NewMemStorage()
	err := repo.LoadFromFile("/nonexistent/path/file.json")
	assert.Error(t, err)
}

func TestMemStorage_SaveToFile_InvalidPath(t *testing.T) {
	repo := NewMemStorage()
	err := repo.SaveToFile("/nonexistent/path/file.json")
	assert.Error(t, err)
}