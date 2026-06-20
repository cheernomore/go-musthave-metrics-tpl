package repository

import (
	"strconv"
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
)

func benchBatch(n int) []models.Metrics {
	batch := make([]models.Metrics, 0, n)
	for i := 0; i < n; i++ {
		v := float64(i) * 1.5
		batch = append(batch, models.Metrics{
			ID:    "Gauge" + strconv.Itoa(i),
			MType: models.Gauge,
			Value: &v,
		})
	}
	return batch
}

func BenchmarkMemStorage_SaveBatch(b *testing.B) {
	batch := benchBatch(29)
	repo := NewMemStorage()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.SaveBatch(batch)
	}
}

func BenchmarkMemStorage_Save(b *testing.B) {
	repo := NewMemStorage()
	v := 42.0
	m := models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Save(m)
	}
}

func BenchmarkMemStorage_FindAll(b *testing.B) {
	repo := NewMemStorage()
	_ = repo.SaveBatch(benchBatch(29))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.FindAll()
	}
}
