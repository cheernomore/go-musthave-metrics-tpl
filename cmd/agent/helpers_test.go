package main

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"syscall"
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToModel(t *testing.T) {
	tests := []struct {
		name      string
		metric    Metric
		wantOK    bool
		wantGauge bool
	}{
		{"float64 gauge", Metric{"Alloc", float64(1.5), "gauge"}, true, true},
		{"int64 counter", Metric{"PollCount", int64(7), "counter"}, true, false},
		{"uint64 gauge", Metric{"HeapAlloc", uint64(100), "gauge"}, true, true},
		{"uint64 counter", Metric{"Cnt", uint64(100), "counter"}, true, false},
		{"uint32 gauge", Metric{"U32", uint32(10), "gauge"}, true, true},
		{"uint8 counter", Metric{"U8", uint8(3), "counter"}, true, false},
		{"unsupported type", Metric{"Bad", "string-value", "gauge"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, ok := convertToModel(tt.metric)
			require.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				return
			}
			assert.Equal(t, tt.metric.Name, m.ID)
			if tt.wantGauge {
				assert.NotNil(t, m.Value)
				assert.Nil(t, m.Delta)
			} else {
				assert.NotNil(t, m.Delta)
				assert.Nil(t, m.Value)
			}
		})
	}
}

func TestCompressData(t *testing.T) {
	original := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)

	buf, err := compressData(original)
	require.NoError(t, err)
	assert.Less(t, 0, buf.Len())

	gz, err := gzip.NewReader(buf)
	require.NoError(t, err)
	defer gz.Close()

	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)
}

func TestCalculateHash(t *testing.T) {
	data := []byte("payload")
	key := "secret"

	got := calculateHash(data, key)

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	want := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, want, got)
	assert.Len(t, got, 64)
}

func TestIsRetriableError(t *testing.T) {
	assert.False(t, isRetriableError(nil))
	assert.False(t, isRetriableError(io.EOF))
	assert.True(t, isRetriableError(syscall.ECONNREFUSED))
	assert.True(t, isRetriableError(syscall.ECONNRESET))
}

func TestGetMetricsPooler(t *testing.T) {
	pooler := getMetricsPooler()
	var memStats runtime.MemStats

	first := pooler(&memStats)
	second := pooler(&memStats)

	require.NotEmpty(t, first)

	names := make(map[string]any, len(first))
	for _, m := range first {
		names[m.Name] = m.Value
	}
	assert.Contains(t, names, "Alloc")
	assert.Contains(t, names, "PollCount")

	// PollCount — счётчик, должен инкрементироваться между вызовами.
	assert.Equal(t, int64(1), names["PollCount"])
	for _, m := range second {
		if m.Name == "PollCount" {
			assert.Equal(t, int64(2), m.Value)
		}
	}
}

func TestCollectGopsutilMetrics(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = collectGopsutilMetrics()
	})
}

func TestGetClient(t *testing.T) {
	c := getClient()
	assert.NotNil(t, c)
}

func TestSendMetricsBatch(t *testing.T) {
	key := "secret"

	var gotEncoding, gotHash string
	var gotMetrics []models.Metrics

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEncoding = r.Header.Get("Content-Encoding")
		gotHash = r.Header.Get("HashSHA256")

		gz, err := gzip.NewReader(r.Body)
		require.NoError(t, err)
		defer gz.Close()
		body, err := io.ReadAll(gz)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &gotMetrics))

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	v := 12.5
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &v}}
	client := getClient()

	res, err := SendMetricsBatch(ts.URL, batch, key, &client)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "gzip", gotEncoding)
	assert.NotEmpty(t, gotHash)
	require.Len(t, gotMetrics, 1)
	assert.Equal(t, "Alloc", gotMetrics[0].ID)
}
