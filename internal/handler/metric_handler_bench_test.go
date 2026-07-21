package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
)

// sampleBatch формирует реалистичный набор метрик (как присылает агент).
func sampleBatch(n int) []models.Metrics {
	batch := make([]models.Metrics, 0, n)
	for i := 0; i < n; i++ {
		v := float64(i) * 1.5
		batch = append(batch, models.Metrics{
			ID:    "Gauge" + strconv.Itoa(i),
			MType: models.Gauge,
			Value: &v,
		})
	}
	d := int64(n)
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d})
	return batch
}

func populatedRepo(n int) *repository.MemStorage {
	repo := repository.NewMemStorage()
	_ = repo.SaveBatch(sampleBatch(n))
	return repo
}

func BenchmarkMetricHandler_Updates(b *testing.B) {
	h := NewMetricHandler(repository.NewMemStorage(), nil)
	body, _ := json.Marshal(sampleBatch(29))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Updates(rec, req)
	}
}

func BenchmarkMetricHandler_Index(b *testing.B) {
	h := NewMetricHandler(populatedRepo(29), nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.Index(rec, req)
	}
}

func BenchmarkMetricHandler_Value(b *testing.B) {
	h := NewMetricHandler(populatedRepo(29), nil)
	body, _ := json.Marshal(models.Metrics{ID: "Gauge1", MType: models.Gauge})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Value(rec, req)
	}
}
