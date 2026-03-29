package handler

import (
	"net/http"
	"strconv"
)

type gauge float64
type counter int64
type MemStorage struct {
	Gauges   map[string]gauge
	Counters map[string]counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Gauges:   make(map[string]gauge),
		Counters: make(map[string]counter),
	}
}

var Storage = NewMemStorage()

func UpdateHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Content-Type wrong", http.StatusBadRequest)
		return
	}

	metricType := r.PathValue("metricType")
	metricName := r.PathValue("metricName")
	value := r.PathValue("value")

	if metricName == "" {
		http.Error(w, "Metric name not present", http.StatusNotFound)
		return
	}

	switch metricType {
	case "counter":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			http.Error(w, "Invalid value for counter", http.StatusBadRequest)
			return
		}
		Storage.Counters[metricName] += counter(v)
	case "gauge":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			http.Error(w, "Invalid value for gauge", http.StatusBadRequest)
			return
		}
		Storage.Gauges[metricName] = gauge(v)
	default:
		http.Error(w, "Unknown metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}
