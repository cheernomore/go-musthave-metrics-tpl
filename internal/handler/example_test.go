package handler_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
)

// newTestServer поднимает тестовый сервер со всеми эндпоинтами на in-memory
// хранилище, наполняя его переданными метриками.
func newTestServer(seed ...models.Metrics) *httptest.Server {
	repo := repository.NewMemStorage()
	for _, m := range seed {
		_ = repo.Save(m)
	}
	h := handler.NewMetricHandler(repo, nil)

	r := chi.NewRouter()
	r.Post("/updates/", h.Updates)
	r.Post("/update/{metricType}/{metricName}/{value}", h.Update)
	r.Post("/update", h.UpdateNew)
	r.Post("/value", h.Value)
	r.Get("/value/{metricType}/{metricName}", h.Get)
	r.Get("/", h.Index)
	return httptest.NewServer(r)
}

func gaugeOf(v float64) *float64 { return &v }
func deltaOf(v int64) *int64     { return &v }

// ExampleMetricHandler_Update — приём метрики через параметры URL:
// POST /update/{type}/{name}/{value}.
func ExampleMetricHandler_Update() {
	ts := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/update/gauge/Alloc/123.45", "text/plain", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

// ExampleMetricHandler_UpdateNew — приём одной метрики в формате JSON
// (POST /update). В ответ возвращается сохранённая метрика.
func ExampleMetricHandler_UpdateNew() {
	ts := newTestServer()
	defer ts.Close()

	body := `{"id":"Alloc","type":"gauge","value":123.45}`
	resp, err := http.Post(ts.URL+"/update", "application/json", strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Print(string(out))
	// Output:
	// {"id":"Alloc","type":"gauge","value":123.45}
}

// ExampleMetricHandler_Updates — приём пакета метрик в формате JSON
// (POST /updates/).
func ExampleMetricHandler_Updates() {
	ts := newTestServer()
	defer ts.Close()

	body := `[{"id":"Alloc","type":"gauge","value":1.5},{"id":"PollCount","type":"counter","delta":3}]`
	resp, err := http.Post(ts.URL+"/updates/", "application/json", strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

// ExampleMetricHandler_Value — запрос значения метрики в формате JSON
// (POST /value).
func ExampleMetricHandler_Value() {
	ts := newTestServer(models.Metrics{ID: "PollCount", MType: models.Counter, Delta: deltaOf(5)})
	defer ts.Close()

	body := `{"id":"PollCount","type":"counter"}`
	resp, err := http.Post(ts.URL+"/value", "application/json", strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Print(string(out))
	// Output:
	// {"id":"PollCount","type":"counter","delta":5}
}

// ExampleMetricHandler_Get — получение значения метрики в виде текста
// (GET /value/{type}/{name}).
func ExampleMetricHandler_Get() {
	ts := newTestServer(models.Metrics{ID: "Alloc", MType: models.Gauge, Value: gaugeOf(42)})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/value/gauge/Alloc")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
	// Output:
	// 42
}
