package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type Metric struct {
	Name  string
	Value any
	Type  string
}

type SendResult struct {
	StatusCode int
	Header     string
}

//	комментарий костыль для проверки actions
func main() {
	parseFlags()
	var m runtime.MemStats
	var metrics []Metric

	var mu sync.Mutex

	metricsPooler := getMetricsPooler()
	client := getClient()

	pollInterval := time.Duration(flagPollInterval) * time.Second
	reportInterval := time.Duration(flagReportInterval) * time.Second

	go func() {
		ticker := time.NewTicker(pollInterval)
		for range ticker.C {
			mu.Lock()
			metrics = metricsPooler(&m)
			mu.Unlock()
		}
	}()

	updateMetricsTicker := time.NewTicker(reportInterval)
	defer updateMetricsTicker.Stop()

	for range updateMetricsTicker.C {
		mu.Lock()
		localMetrics := make([]Metric, len(metrics))
		copy(localMetrics, metrics)
		mu.Unlock()

		for _, metric := range metrics {
			_, err := SendMetrics("http://"+flagAddressPort+"/update/", metric.Type, metric.Name, metric.Value, &client)
			if err != nil {
				fmt.Println("Error request")
			}
		}
	}
}

func SendMetrics(baseURL string, metricType string, metricName string, value any, client *http.Client) (SendResult, error) {
	url := baseURL + metricType + "/" + metricName + "/" + valueToString(metricType, value)
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Printf("Error create request: %s", err)
	}

	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Error with request: %s", err)
	}

	defer response.Body.Close()

	fmt.Printf("Sent: %s, Status: %s\n", url, response.Status)
	return SendResult{
		StatusCode: response.StatusCode,
		Header:     response.Header.Get("Content-Type"),
	}, nil
}

func getClient() http.Client {
	return http.Client{}
}

func getMetricsPooler() func(m *runtime.MemStats) []Metric {
	var pollCount int64 = 1

	return func(m *runtime.MemStats) []Metric {
		fmt.Println("-----METRICS UPDATED-----")
		runtime.ReadMemStats(m)

		if pollCount == 5 {
			pollCount = 0
		}
		pollCount++

		return []Metric{
			{"Alloc", m.Alloc, "gauge"},
			{"BuckHashSys", m.BuckHashSys, "gauge"},
			{"Frees", m.Frees, "gauge"},
			{"GCCPUFraction", m.GCCPUFraction, "gauge"},
			{"GCSys", m.GCSys, "gauge"},
			{"HeapAlloc", m.HeapAlloc, "gauge"},
			{"HeapIdle", m.HeapIdle, "gauge"},
			{"HeapInuse", m.HeapInuse, "gauge"},
			{"HeapObjects", m.HeapObjects, "gauge"},
			{"HeapReleased", m.HeapReleased, "gauge"},
			{"HeapSys", m.HeapSys, "gauge"},
			{"LastGC", m.LastGC, "gauge"},
			{"Lookups", m.Lookups, "gauge"},
			{"MCacheInuse", m.MCacheInuse, "gauge"},
			{"MCacheSys", m.MCacheSys, "gauge"},
			{"MSpanInuse", m.MSpanInuse, "gauge"},
			{"MSpanSys", m.MSpanSys, "gauge"},
			{"Mallocs", m.Mallocs, "gauge"},
			{"NextGC", m.NextGC, "gauge"},
			{"NumForcedGC", m.NumForcedGC, "gauge"},
			{"NumGC", m.NumGC, "gauge"},
			{"OtherSys", m.OtherSys, "gauge"},
			{"PauseTotalNs", m.PauseTotalNs, "gauge"},
			{"StackInuse", m.StackInuse, "gauge"},
			{"StackSys", m.StackSys, "gauge"},
			{"Sys", m.Sys, "gauge"},
			{"TotalAlloc", m.TotalAlloc, "gauge"},
			{"PollCount", pollCount, "counter"},
			{"RandomValue", rand.Float64() * 1000, "gauge"},
		}
	}
}

func valueToString(metricType string, v any) string {
	var f float64
	var i int64

	if metricType == "counter" {
		switch t := v.(type) {
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

	return strconv.FormatFloat(f, 'f', -1, 64)
}
