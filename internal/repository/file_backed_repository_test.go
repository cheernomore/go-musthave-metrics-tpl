package repository

import (
	"path/filepath"
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileBackedRepository_SaveAndPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := NewFileBackedRepository(NewMemStorage(), path)

	v := 3.14
	require.NoError(t, repo.Save(models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v}))

	got, err := repo.Find("Alloc", models.Gauge)
	require.NoError(t, err)
	require.NotNil(t, got.Value)
	assert.Equal(t, v, *got.Value)

	// данные должны быть сохранены в файл — проверяем загрузкой в новое хранилище
	reloaded := NewMemStorage()
	require.NoError(t, reloaded.LoadFromFile(path))
	got2, err := reloaded.Find("Alloc", models.Gauge)
	require.NoError(t, err)
	require.NotNil(t, got2.Value)
	assert.Equal(t, v, *got2.Value)
}

func TestFileBackedRepository_SaveBatchAndFindAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := NewFileBackedRepository(NewMemStorage(), path)

	d := int64(5)
	v := 1.0
	batch := []models.Metrics{
		{ID: "PollCount", MType: models.Counter, Delta: &d},
		{ID: "Gauge0", MType: models.Gauge, Value: &v},
	}
	require.NoError(t, repo.SaveBatch(batch))

	all, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}
