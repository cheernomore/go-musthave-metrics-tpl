package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/buildinfo"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/crypto"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/grpcapi"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/netutil"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/retry"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Переменные сборки задаются через -ldflags при компиляции, например:
// go build -ldflags "-X main.buildVersion=v1.0.0".
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// Metric — собранная агентом метрика во внутреннем представлении до
// преобразования в модель отправки.
type Metric struct {
	// Name — имя метрики.
	Name string
	// Value — значение метрики (числовой тип, зависящий от метрики).
	Value any
	// Type — тип метрики: "gauge" или "counter".
	Type string
}

// SendResult — результат отправки пакета метрик на сервер.
type SendResult struct {
	// StatusCode — HTTP-код ответа сервера.
	StatusCode int
	// Header — значение заголовка Content-Type ответа.
	Header string
}

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	cfg := parseFlags()

	var pubKey *rsa.PublicKey
	if cfg.CryptoKey != "" {
		var err error
		pubKey, err = crypto.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			log.Fatalf("не удалось загрузить публичный ключ: %v", err)
		}
		fmt.Println("асимметричное шифрование включено")
	}

	// IP-адрес хоста агента передаётся серверу в заголовке X-Real-IP.
	var realIP string
	if ip, err := netutil.LocalIP(); err != nil {
		log.Printf("не удалось определить IP-адрес агента: %v", err)
	} else {
		realIP = ip.String()
	}

	sendOpts := SendOptions{Key: cfg.Key, PubKey: pubKey, RealIP: realIP}

	// Если задан адрес gRPC-сервера, метрики отправляются по gRPC.
	var grpcClient *grpcapi.Client
	if cfg.GRPCAddress != "" {
		var err error
		grpcClient, err = grpcapi.NewClient(cfg.GRPCAddress, realIP)
		if err != nil {
			log.Fatalf("не удалось создать gRPC-клиент: %v", err)
		}
		defer grpcClient.Close()
		fmt.Printf("отправка метрик по gRPC на %s\n", cfg.GRPCAddress)
	}

	// Контекст, отменяемый по сигналам штатного завершения.
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	var mu sync.Mutex
	var runtimeMetrics []Metric
	var extraMetrics []Metric

	metricsPooler := getMetricsPooler()
	client := getClient()

	pollInterval := time.Duration(cfg.PollInterval) * time.Second
	reportInterval := time.Duration(cfg.ReportInterval) * time.Second

	var memStats runtime.MemStats
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				runtimeMetrics = metricsPooler(&memStats)
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				extraMetrics = collectGopsutilMetrics()
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	jobs := make(chan []models.Metrics, 10)

	rateLimit := cfg.RateLimit
	if rateLimit < 1 {
		rateLimit = 1
	}
	// sendBatch отправляет пакет метрик выбранным транспортом. Контекст
	// намеренно не связан с сигнальным: при завершении агента досылка
	// накопленных метрик должна успеть выполниться.
	sendBatch := func(batch []models.Metrics) error {
		if grpcClient != nil {
			sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return grpcClient.Send(sendCtx, batch)
		}
		_, err := SendMetricsBatch("http://"+cfg.Address+"/updates/", batch, sendOpts, &client)
		return err
	}

	var workers sync.WaitGroup
	for range rateLimit {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for batch := range jobs {
				err := retry.WithRetry(func() error {
					return sendBatch(batch)
				}, isRetriableError)
				if err != nil {
					fmt.Printf("Failed to send metrics batch after retries: %v\n", err)
				}
			}
		}()
	}

	// snapshot возвращает копию собранных на текущий момент метрик.
	snapshot := func() []Metric {
		mu.Lock()
		defer mu.Unlock()
		all := make([]Metric, 0, len(runtimeMetrics)+len(extraMetrics))
		all = append(all, runtimeMetrics...)
		all = append(all, extraMetrics...)
		return all
	}
	// enqueue формирует пакет из текущих метрик и ставит его в очередь отправки.
	enqueue := func() {
		if batch := buildBatch(snapshot()); len(batch) > 0 {
			jobs <- batch
		}
	}

	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

loop:
	for {
		select {
		case <-reportTicker.C:
			enqueue()
		case <-ctx.Done():
			// Данные, находящиеся в обработке на момент сигнала, должны быть
			// отправлены на сервер.
			fmt.Println("получен сигнал завершения, досылаем метрики")
			enqueue()
			break loop
		}
	}

	// Закрываем очередь и ждём, пока воркеры отправят оставшиеся пакеты.
	close(jobs)
	workers.Wait()
	fmt.Println("агент остановлен")
}

// buildBatch преобразует собранные метрики в модель отправки.
func buildBatch(metrics []Metric) []models.Metrics {
	batch := make([]models.Metrics, 0, len(metrics))
	for _, metric := range metrics {
		if payload, ok := convertToModel(metric); ok {
			batch = append(batch, payload)
		}
	}
	return batch
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

// SendOptions — параметры отправки пакета метрик.
type SendOptions struct {
	// Key — ключ подписи HMAC-SHA256; пусто — запрос не подписывается.
	Key string
	// PubKey — публичный RSA-ключ; nil — тело не шифруется.
	PubKey *rsa.PublicKey
	// RealIP — IP-адрес хоста агента для заголовка X-Real-IP;
	// пусто — заголовок не добавляется.
	RealIP string
}

// SendMetricsBatch отправляет пакет метрик POST-запросом на url. Тело
// сериализуется в JSON и сжимается gzip; при непустом opts.Key добавляется
// подпись HMAC-SHA256 в заголовке HashSHA256. Если задан opts.PubKey, сжатое
// тело дополнительно шифруется RSA-ключом, а запрос помечается заголовком
// crypto.EncryptedHeader. IP агента передаётся в заголовке X-Real-IP.
func SendMetricsBatch(url string, metrics []models.Metrics, opts SendOptions, client *http.Client) (SendResult, error) {
	body, err := json.Marshal(metrics)
	if err != nil {
		return SendResult{}, err
	}

	buf, err := compressData(body)
	if err != nil {
		return SendResult{}, err
	}

	payload := buf.Bytes()
	encrypted := false
	if opts.PubKey != nil {
		payload, err = crypto.Encrypt(opts.PubKey, payload)
		if err != nil {
			return SendResult{}, fmt.Errorf("encrypt error: %w", err)
		}
		encrypted = true
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return SendResult{}, fmt.Errorf("request creation error: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")
	if encrypted {
		request.Header.Set(crypto.EncryptedHeader, "1")
	}
	if opts.RealIP != "" {
		request.Header.Set("X-Real-IP", opts.RealIP)
	}

	if opts.Key != "" {
		hash := calculateHash(body, opts.Key)
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
