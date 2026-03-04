package handler

import (
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi"
	"html/template"
	"net/http"
	"strconv"
)

type MetricHandler struct {
	repo repository.MetricsRepository
}

func NewMetricHandler(repo repository.MetricsRepository) *MetricHandler {
	return &MetricHandler{repo: repo}
}

func (h *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	metricTypeURLParam := chi.URLParam(r, "metricType")
	metricNameURLParam := chi.URLParam(r, "metricName")
	valueURLParam := chi.URLParam(r, "value")

	if metricNameURLParam == "" {
		http.Error(w, "Metric name not present", http.StatusNotFound)
		return
	}

	if metricTypeURLParam == models.Counter {
		parsedValue, err := strconv.ParseInt(valueURLParam, 10, 64)
		if err != nil {
			http.Error(w, "invalid value for counter", http.StatusBadRequest)
			return
		}
		m = models.Metrics{
			ID:    metricNameURLParam,
			MType: metricTypeURLParam,
			Delta: &parsedValue,
			Value: nil,
			Hash:  "",
		}
	} else if metricTypeURLParam == models.Gauge {
		parsedValue, err := strconv.ParseFloat(valueURLParam, 64)
		if err != nil {
			http.Error(w, "invalid value for gauge", http.StatusBadRequest)
			return
		}
		m = models.Metrics{
			ID:    metricNameURLParam,
			MType: metricTypeURLParam,
			Delta: nil,
			Value: &parsedValue,
			Hash:  "",
		}
	} else {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	err := h.repo.Save(m)
	if err != nil {
		http.Error(w, "problem with save metric to repo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) Get(w http.ResponseWriter, r *http.Request) {
	metricTypeURLParam := chi.URLParam(r, "metricType")
	metricNameURLParam := chi.URLParam(r, "metricName")

	if metricNameURLParam == "" {
		http.Error(w, "metric name not present", http.StatusNotFound)
		return
	}

	m, err := h.repo.Find(metricNameURLParam, metricTypeURLParam)
	if err != nil {
		http.Error(w, "metric not found", http.StatusNotFound)
		return
	}

	if m.MType == models.Counter {
		valueStr := strconv.FormatInt(*m.Delta, 10)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(valueStr))
	} else {
		valueStr := strconv.FormatFloat(*m.Value, 'f', -1, 64)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(valueStr))
	}
}

func (h *MetricHandler) Index(w http.ResponseWriter, r *http.Request) {
	const htmlTemplate = `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Metrics Monitoring</title>
		</head>
		<body>
			<h1>Current Metrics</h1>
			<h2>Counters</h2>
			<ul>
			{{range $metric := .}}
				 {{if eq $metric.MType "counter"}}
					<li>{{$metric.ID}}: {{$metric.Delta}}</li>
				{{end}}
			{{end}}
			</ul>
			<h2>Gauges</h2>
			<ul>
			{{range $metric := .}}
				 {{if eq $metric.MType "gauge"}}
					<li>{{$metric.ID}}: {{$metric.Value}}</li>
				{{end}}
			{{end}}
			</ul>
		</body>
		</html>`

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	m, err := h.repo.FindAll()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(w, m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
