package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/retry"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"sync"
	"syscall"
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

		if len(localMetrics) == 0 {
			continue
		}

		batch := make([]models.Metrics, 0, len(localMetrics))
		for _, metric := range localMetrics {
			payload := models.Metrics{
				ID:    metric.Name,
				MType: metric.Type,
			}

			switch v := metric.Value.(type) {
			case float64:
				payload.Value = &v
			case int64:
				payload.Delta = &v
			case uint64:
				if metric.Type == "gauge" {
					floatVal := float64(v)
					payload.Value = &floatVal
				} else {
					intVal := int64(v)
					payload.Delta = &intVal
				}
			case uint32:
				if metric.Type == "gauge" {
					floatVal := float64(v)
					payload.Value = &floatVal
				} else {
					intVal := int64(v)
					payload.Delta = &intVal
				}
			case uint16:
				if metric.Type == "gauge" {
					floatVal := float64(v)
					payload.Value = &floatVal
				} else {
					intVal := int64(v)
					payload.Delta = &intVal
				}
			case uint8:
				if metric.Type == "gauge" {
					floatVal := float64(v)
					payload.Value = &floatVal
				} else {
					intVal := int64(v)
					payload.Delta = &intVal
				}
			}

			batch = append(batch, payload)
		}

		// Отправка метрик с retry логикой
		err := retry.WithRetry(func() error {
			_, err := SendMetricsBatch("http://"+flagAddressPort+"/updates/", batch, &client)
			return err
		}, isRetriableError)

		if err != nil {
			fmt.Printf("Failed to send metrics batch after retries: %v\n", err)
		}
	}
}

func SendMetricsBatch(url string, metrics []models.Metrics, client *http.Client) (SendResult, error) {
	body, err := json.Marshal(metrics)
	if err != nil {
		return SendResult{}, err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return SendResult{}, fmt.Errorf("gzip compression error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return SendResult{}, fmt.Errorf("gzip close error: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return SendResult{}, fmt.Errorf("request creation error: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")

	response, err := client.Do(request)
	if err != nil {
		return SendResult{}, fmt.Errorf("request execution error: %w", err)
	}
	defer response.Body.Close()

	fmt.Printf("Sent batch (gzipped) to %s, Status: %s\n", url, response.Status)

	return SendResult{
		StatusCode: response.StatusCode,
		Header:     response.Header.Get("Content-Type"),
	}, nil
}

func SendMetrics(url string, metrics models.Metrics, client *http.Client) (SendResult, error) {
	body, err := json.Marshal(metrics)
	if err != nil {
		return SendResult{}, err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return SendResult{}, fmt.Errorf("gzip compression error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return SendResult{}, fmt.Errorf("gzip close error: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return SendResult{}, fmt.Errorf("request creation error: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")

	response, err := client.Do(request)
	if err != nil {
		return SendResult{}, fmt.Errorf("request execution error: %w", err)
	}
	defer response.Body.Close()

	fmt.Printf("Sent (gzipped): %s, Status: %s\n", url, response.Status)

	return SendResult{
		StatusCode: response.StatusCode,
		Header:     response.Header.Get("Content-Type"),
	}, nil
}

func getClient() http.Client {
	return http.Client{}
}

// isRetriableError определяет, является ли ошибка временной (retriable)
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Проверка на таймауты
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Проверка на конкретные системные ошибки соединения
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}

	// Проверка на net.OpError для детального анализа
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Повторяем для вложенных ошибок
		return isRetriableError(opErr.Err)
	}

	// DNS ошибки с таймаутом
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsTimeout {
		return true
	}

	return false
}

func getMetricsPooler() func(m *runtime.MemStats) []Metric {
	var pollCount int64 = 0

	return func(m *runtime.MemStats) []Metric {
		fmt.Println("-----METRICS UPDATED-----")
		runtime.ReadMemStats(m)

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
