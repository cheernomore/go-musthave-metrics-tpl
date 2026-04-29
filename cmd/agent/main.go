package main

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/retry"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
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

	var mu sync.Mutex
	var runtimeMetrics []Metric
	var extraMetrics []Metric

	metricsPooler := getMetricsPooler()
	client := getClient()

	pollInterval := time.Duration(flagPollInterval) * time.Second
	reportInterval := time.Duration(flagReportInterval) * time.Second

	var memStats runtime.MemStats
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			runtimeMetrics = metricsPooler(&memStats)
			mu.Unlock()
		}
	}()

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			extraMetrics = collectGopsutilMetrics()
			mu.Unlock()
		}
	}()

	jobs := make(chan models.Metrics, 1000)

	rateLimit := flagRateLimit
	if rateLimit < 1 {
		rateLimit = 1
	}
	for range rateLimit {
		go func() {
			for job := range jobs {
				err := retry.WithRetry(func() error {
					_, err := SendMetrics("http://"+flagAddressPort+"/update/", job, &client)
					return err
				}, isRetriableError)
				if err != nil {
					fmt.Printf("Failed to send metric after retries: %v\n", err)
				}
			}
		}()
	}

	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	for range reportTicker.C {
		mu.Lock()
		all := make([]Metric, 0, len(runtimeMetrics)+len(extraMetrics))
		all = append(all, runtimeMetrics...)
		all = append(all, extraMetrics...)
		mu.Unlock()

		if len(all) == 0 {
			continue
		}

		for _, metric := range all {
			payload, ok := convertToModel(metric)
			if !ok {
				continue
			}
			jobs <- payload
		}
	}
}

func convertToModel(metric Metric) (models.Metrics, bool) {
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
			f := float64(v)
			payload.Value = &f
		} else {
			i := int64(v)
			payload.Delta = &i
		}
	case uint32:
		if metric.Type == "gauge" {
			f := float64(v)
			payload.Value = &f
		} else {
			i := int64(v)
			payload.Delta = &i
		}
	case uint16:
		if metric.Type == "gauge" {
			f := float64(v)
			payload.Value = &f
		} else {
			i := int64(v)
			payload.Delta = &i
		}
	case uint8:
		if metric.Type == "gauge" {
			f := float64(v)
			payload.Value = &f
		} else {
			i := int64(v)
			payload.Delta = &i
		}
	default:
		return models.Metrics{}, false
	}

	return payload, true
}

func collectGopsutilMetrics() []Metric {
	var result []Metric

	vmStat, err := mem.VirtualMemory()
	if err == nil {
		result = append(result,
			Metric{"TotalMemory", float64(vmStat.Total), "gauge"},
			Metric{"FreeMemory", float64(vmStat.Free), "gauge"},
		)
	}

	cpuPercents, err := cpu.Percent(0, true)
	if err == nil {
		for i, p := range cpuPercents {
			result = append(result, Metric{
				fmt.Sprintf("CPUutilization%d", i+1),
				p,
				"gauge",
			})
		}
	}

	return result
}

func compressData(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("gzip compression error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close error: %w", err)
	}
	return &buf, nil
}

func calculateHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func SendMetricsBatch(url string, metrics []models.Metrics, client *http.Client) (SendResult, error) {
	body, err := json.Marshal(metrics)
	if err != nil {
		return SendResult{}, err
	}

	buf, err := compressData(body)
	if err != nil {
		return SendResult{}, err
	}

	request, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return SendResult{}, fmt.Errorf("request creation error: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")

	if flagKey != "" {
		hash := calculateHash(body, flagKey)
		request.Header.Set("HashSHA256", hash)
	}

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

	buf, err := compressData(body)
	if err != nil {
		return SendResult{}, err
	}

	request, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return SendResult{}, fmt.Errorf("request creation error: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")

	if flagKey != "" {
		hash := calculateHash(body, flagKey)
		request.Header.Set("HashSHA256", hash)
	}

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

func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return isRetriableError(opErr.Err)
	}

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
