// Package handler содержит HTTP-обработчики API сервера сбора метрик.
package handler

import (
	"encoding/json"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/audit"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/logger"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MetricHandler обслуживает HTTP-эндпоинты приёма и выдачи метрик. Хранение
// делегируется repo, а после успешной записи события рассылаются через auditor.
type MetricHandler struct {
	repo    repository.MetricsRepository
	auditor *audit.Subject
}

// NewMetricHandler создаёт обработчик поверх хранилища repo. Параметр auditor
// может быть nil — в этом случае аудит запросов отключён.
func NewMetricHandler(repo repository.MetricsRepository, auditor *audit.Subject) *MetricHandler {
	return &MetricHandler{
		repo:    repo,
		auditor: auditor,
	}
}

// audit формирует событие аудита по обработанным метрикам и рассылает его
// всем приёмникам. Если аудит отключён, вызов ничего не делает.
func (h *MetricHandler) audit(r *http.Request, metricNames []string) {
	if !h.auditor.HasObservers() {
		return
	}

	h.auditor.Notify(audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: clientIP(r),
	})
}

// clientIP определяет IP-адрес входящего запроса, учитывая заголовки прокси.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.IndexByte(fwd, ','); i >= 0 {
			return strings.TrimSpace(fwd[:i])
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Updates обрабатывает POST /updates/ — приём пакета метрик в формате JSON.
// Тело запроса — массив метрик. Возвращает 200 при успехе, 400 при некорректном
// JSON и 500 при ошибке сохранения.
func (h *MetricHandler) Updates(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&metrics); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.repo.SaveBatch(metrics); err != nil {
		http.Error(w, "problem with save metrics to repo", http.StatusInternalServerError)
		return
	}

	if h.auditor.HasObservers() {
		names := make([]string, 0, len(metrics))
		for _, m := range metrics {
			names = append(names, m.ID)
		}
		h.audit(r, names)
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateNew обрабатывает POST /update — приём одной метрики в формате JSON.
// В ответ возвращается сохранённая метрика (с актуальным значением) в JSON.
func (h *MetricHandler) UpdateNew(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics

	logger.Log.Debug("decoding request")
	var req models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch req.MType {
	case models.Counter:
		m = models.Metrics{
			ID:    req.ID,
			MType: req.MType,
			Delta: req.Delta,
			Value: nil,
		}
	case models.Gauge:
		m = models.Metrics{
			ID:    req.ID,
			MType: req.MType,
			Delta: nil,
			Value: req.Value,
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	err := h.repo.Save(m)
	if err != nil {
		http.Error(w, "problem with save metric to repo", http.StatusInternalServerError)
		return
	}

	saved, err := h.repo.Find(m.ID, m.MType)
	if err != nil {
		http.Error(w, "problem retrieving saved metric", http.StatusInternalServerError)
		return
	}

	if h.auditor.HasObservers() {
		h.audit(r, []string{m.ID})
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(saved); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

// Value обрабатывает POST /value — запрос значения метрики по её имени и типу,
// переданным в теле в формате JSON. Возвращает метрику в JSON или 404.
func (h *MetricHandler) Value(w http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("decoding request")
	var req models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	find, err := h.repo.Find(req.ID, req.MType)
	if err != nil {
		http.Error(w, "Metric not present", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(find); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

// Update обрабатывает POST /update/{metricType}/{metricName}/{value} —
// приём одной метрики через параметры URL. Возвращает 200 при успехе,
// 400 при некорректном значении/типе и 404 при отсутствии имени.
func (h *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	metricTypeURLParam := chi.URLParam(r, "metricType")
	metricNameURLParam := chi.URLParam(r, "metricName")
	valueURLParam := chi.URLParam(r, "value")

	if metricNameURLParam == "" {
		http.Error(w, "Metric name not present", http.StatusNotFound)
		return
	}

	switch metricTypeURLParam {
	case models.Counter:
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
		}
	case models.Gauge:
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
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	err := h.repo.Save(m)
	if err != nil {
		http.Error(w, "problem with save metric to repo", http.StatusInternalServerError)
		return
	}

	if h.auditor.HasObservers() {
		h.audit(r, []string{m.ID})
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// Get обрабатывает GET /value/{metricType}/{metricName} — выдачу значения
// метрики в виде текста. Возвращает 404, если метрика не найдена.
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

// indexTemplate компилируется один раз при инициализации пакета, чтобы не
// тратить аллокации на разбор шаблона при каждом запросе к Index.
var indexTemplate = template.Must(template.New("index").Parse(`
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
		</html>`))

// Index обрабатывает GET / — HTML-страницу со списком всех метрик,
// сгруппированных по типам (counter и gauge).
func (h *MetricHandler) Index(w http.ResponseWriter, r *http.Request) {
	m, err := h.repo.FindAll()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTemplate.Execute(w, m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
