package repository

import (
	"errors"
	"path/filepath"
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errInner = errors.New("сбой вложенного хранилища")

// failingRepo — хранилище, всегда возвращающее ошибку.
type failingRepo struct{}

func (failingRepo) Save(models.Metrics) error        { return errInner }
func (failingRepo) SaveBatch([]models.Metrics) error { return errInner }
func (failingRepo) Find(string, string) (models.Metrics, error) {
	return models.Metrics{}, errInner
}
func (failingRepo) FindAll() ([]models.Metrics, error) { return nil, errInner }

func TestFileBackedRepository_InnerErrors(t *testing.T) {
	repo := NewFileBackedRepository(failingRepo{}, filepath.Join(t.TempDir(), "m.json"))

	assert.ErrorIs(t, repo.Save(models.Metrics{}), errInner)
	assert.ErrorIs(t, repo.SaveBatch(nil), errInner)

	_, err := repo.Find("X", models.Gauge)
	assert.ErrorIs(t, err, errInner)

	_, err = repo.FindAll()
	assert.ErrorIs(t, err, errInner)
}

func TestFileBackedRepository_SaveToFileError(t *testing.T) {
	// Каталог не существует — запись в файл завершится ошибкой.
	badPath := filepath.Join(t.TempDir(), "нет-каталога", "m.json")
	repo := NewFileBackedRepository(NewMemStorage(), badPath)

	v := 1.5
	metric := models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v}

	require.Error(t, repo.Save(metric))
	require.Error(t, repo.SaveBatch([]models.Metrics{metric}))
}
