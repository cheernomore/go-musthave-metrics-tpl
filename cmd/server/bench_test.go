package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/cheernomore/go-musthave-metrics-tpl/internal/audit"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
)

// benchRouter собирает роутер с полной цепочкой middleware,
// как в run(), но без БД — на in-memory хранилище.
func benchRouter(repo repository.MetricsRepository) http.Handler {
	metricHandler := handler.NewMetricHandler(repo, audit.NewSubject())

	r := chi.NewRouter()
	r.Use(GzipMiddleware)
	r.Use(HashValidationMiddleware(""))
	r.Use(HashResponseMiddleware(""))

	r.Post("/updates/", metricHandler.Updates)
	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)
	r.Post("/update", metricHandler.UpdateNew)
	r.Post("/value", metricHandler.Value)
	r.Get("/value/{metricType}/{metricName}", metricHandler.Get)
	r.Get("/", metricHandler.Index)
	return r
}

func benchBatchBody(n int) []byte {
	batch := make([]models.Metrics, 0, n+1)
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
	body, _ := json.Marshal(batch)
	return body
}

// BenchmarkServerHotPath эмулирует горячий путь сервера: приём батча метрик
// и отдачу HTML-страницы со списком метрик. Используется для снятия профиля
// потребления памяти (-memprofile).
func BenchmarkServerHotPath(b *testing.B) {
	repo := repository.NewMemStorage()
	router := benchRouter(repo)
	body := benchBatchBody(29)

	// прогрев: наполняем хранилище реалистичными данными
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body)))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updReq := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		updReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(httptest.NewRecorder(), updReq)

		idxReq := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(httptest.NewRecorder(), idxReq)
	}
}
