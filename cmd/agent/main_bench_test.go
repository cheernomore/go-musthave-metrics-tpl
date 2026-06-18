package main

import (
	"encoding/json"
	"runtime"
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
)

func BenchmarkConvertToModel(b *testing.B) {
	metrics := []Metric{
		{"Alloc", uint64(123456), "gauge"},
		{"PollCount", int64(7), "counter"},
		{"RandomValue", 3.14, "gauge"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, m := range metrics {
			_, _ = convertToModel(m)
		}
	}
}

func BenchmarkCompressData(b *testing.B) {
	pooler := getMetricsPooler()
	var memStats runtime.MemStats
	all := pooler(&memStats)

	batch := make([]models.Metrics, 0, len(all))
	for _, m := range all {
		if payload, ok := convertToModel(m); ok {
			batch = append(batch, payload)
		}
	}
	body, _ := json.Marshal(batch)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := compressData(body); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetMetricsPooler(b *testing.B) {
	pooler := getMetricsPooler()
	var memStats runtime.MemStats

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pooler(&memStats)
	}
}
