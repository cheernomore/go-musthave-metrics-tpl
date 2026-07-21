package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errRepo = errors.New("сбой хранилища")

// mockRepo делегирует работу in-memory хранилищу, но позволяет подменить
// результат любой операции ошибкой.
type mockRepo struct {
	inner        *repository.MemStorage
	saveErr      error
	saveBatchErr error
	findErr      error
	findAllErr   error
}

func newMockRepo() *mockRepo {
	return &mockRepo{inner: repository.NewMemStorage()}
}

func (m *mockRepo) Save(metric models.Metrics) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	return m.inner.Save(metric)
}

func (m *mockRepo) SaveBatch(metrics []models.Metrics) error {
	if m.saveBatchErr != nil {
		return m.saveBatchErr
	}
	return m.inner.SaveBatch(metrics)
}

func (m *mockRepo) Find(id, metricType string) (models.Metrics, error) {
	if m.findErr != nil {
		return models.Metrics{}, m.findErr
	}
	return m.inner.Find(id, metricType)
}

func (m *mockRepo) FindAll() ([]models.Metrics, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	return m.inner.FindAll()
}

func post(t *testing.T, h http.HandlerFunc, target string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestUpdates_ErrorPaths(t *testing.T) {
	t.Run("некорректный JSON", func(t *testing.T) {
		h := NewMetricHandler(newMockRepo(), nil)
		rec := post(t, h.Updates, "/updates/", "{не json")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("ошибка сохранения", func(t *testing.T) {
		repo := newMockRepo()
		repo.saveBatchErr = errRepo
		h := NewMetricHandler(repo, nil)

		v := 1.0
		body, _ := json.Marshal([]models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &v}})
		rec := post(t, h.Updates, "/updates/", string(body))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestUpdateNew_ErrorPaths(t *testing.T) {
	t.Run("некорректный JSON", func(t *testing.T) {
		h := NewMetricHandler(newMockRepo(), nil)
		rec := post(t, h.UpdateNew, "/update", "{не json")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("неизвестный тип метрики", func(t *testing.T) {
		h := NewMetricHandler(newMockRepo(), nil)
		rec := post(t, h.UpdateNew, "/update", `{"id":"X","type":"unknown"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("ошибка сохранения", func(t *testing.T) {
		repo := newMockRepo()
		repo.saveErr = errRepo
		h := NewMetricHandler(repo, nil)
		rec := post(t, h.UpdateNew, "/update", `{"id":"Alloc","type":"gauge","value":1.5}`)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("ошибка чтения сохранённой метрики", func(t *testing.T) {
		repo := newMockRepo()
		repo.findErr = errRepo
		h := NewMetricHandler(repo, nil)
		rec := post(t, h.UpdateNew, "/update", `{"id":"Alloc","type":"gauge","value":1.5}`)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestValue_BadJSON(t *testing.T) {
	h := NewMetricHandler(newMockRepo(), nil)
	rec := post(t, h.Value, "/value", "{не json")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdate_URLParamErrors(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   int
	}{
		{"некорректное значение counter", "/update/counter/PollCount/abc", http.StatusBadRequest},
		{"некорректное значение gauge", "/update/gauge/Alloc/abc", http.StatusBadRequest},
		{"неизвестный тип", "/update/unknown/Alloc/1", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMetricHandler(newMockRepo(), nil)
			r := chi.NewRouter()
			r.Post("/update/{metricType}/{metricName}/{value}", h.Update)

			req := httptest.NewRequest(http.MethodPost, tt.target, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

func TestUpdate_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.saveErr = errRepo
	h := NewMetricHandler(repo, nil)

	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{value}", h.Update)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1.5", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGet_CounterAndMissingName(t *testing.T) {
	repo := newMockRepo()
	d := int64(7)
	require.NoError(t, repo.Save(models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d}))
	h := NewMetricHandler(repo, nil)

	r := chi.NewRouter()
	r.Get("/value/{metricType}/{metricName}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "7", rec.Body.String())
}

func TestIndex_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.findAllErr = errRepo
	h := NewMetricHandler(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.Index(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestClientIP_MoreCases(t *testing.T) {
	t.Run("X-Forwarded-For без запятой", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ""
		req.Header.Set("X-Forwarded-For", "10.1.1.1")
		assert.Equal(t, "10.1.1.1", clientIP(req))
	})

	t.Run("RemoteAddr без порта", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.2.2.2"
		assert.Equal(t, "10.2.2.2", clientIP(req))
	})
}
