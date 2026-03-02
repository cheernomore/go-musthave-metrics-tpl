package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"html/template"
	"net/http"
	"strconv"
)

type gauge float64
type counter int64
type MemStorage struct {
	Gauges   map[string]gauge
	Counters map[string]counter
}

func NewMemStorage() MemStorage {
	return MemStorage{
		Gauges:   make(map[string]gauge),
		Counters: make(map[string]counter),
	}
}

var Storage = NewMemStorage()

func Update(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	value := chi.URLParam(r, "value")

	if metricName == "" {
		http.Error(w, "Metric name not present", http.StatusNotFound)
		return
	}

	save(metricType, metricName, value, w)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func Get(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	if metricName == "" {
		http.Error(w, "Metric name not present", http.StatusNotFound)
		return
	}

	if metricType == "gauge" {
		val, ok := Storage.Gauges[metricName]
		if ok {
			fmt.Println("Найдено:", val)
		} else {
			http.Error(w, "Ключ не существует", http.StatusNotFound)
		}

		fmt.Println(valueToString(metricType, Storage.Gauges[metricName]))

		w.Write([]byte(valueToString(metricType, Storage.Gauges[metricName])))
	} else {
		val, ok := Storage.Counters[metricName]
		if ok {
			fmt.Println("Найдено:", val)
		} else {
			http.Error(w, "Ключ не существует", http.StatusNotFound)
		}

		w.Write([]byte(valueToString(metricType, Storage.Counters[metricName])))
		w.WriteHeader(http.StatusOK)
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
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
			{{range $name, $value := .Counters}}
				<li>{{$name}}: {{$value}}</li>
			{{end}}
			</ul>
			<h2>Gauges</h2>
			<ul>
			{{range $name, $value := .Gauges}}
				<li>{{$name}}: {{$value}}</li>
			{{end}}
			</ul>
		</body>
		</html>`

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(w, Storage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func save(metricType string, metricName string, value string, w http.ResponseWriter) {
	switch metricType {
	case "counter":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			http.Error(w, "Invalid value for counter", http.StatusBadRequest)
			return
		}
		Storage.Counters[metricName] = counter(v)
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
}

func valueToString(metricType string, v any) string {
	var f float64
	var i int64

	if metricType == "counter" {
		switch t := v.(type) {
		case counter:
			i = int64(t)
		case int64:
			i = t
		case uint64:
			i = int64(t)
		default:
			return "0.00"
		}
		return strconv.FormatInt(i, 10)
	}

	switch t := v.(type) {
	case gauge:
		f = float64(t)
	case float64:
		f = t
	case uint64:
		f = float64(t)
	case uint32:
		f = float64(t)
	case int64:
		f = float64(t)
	default:
		return "0.00" // или обработка ошибки
	}

	return strconv.FormatFloat(f, 'f', 2, 64)
}
