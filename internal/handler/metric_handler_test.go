package handler

import (
	"bytes"
	"encoding/json"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func testJSONRequest(t *testing.T, ts *httptest.Server, method,
	path string, body interface{}) (*http.Response, string) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, ts.URL+path, bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestMetricHandler_Get(t *testing.T) {
	r := chi.NewRouter()
	repo := repository.NewMemStorage()
	metricHandler := NewMetricHandler(repo)

	r.Get("/value/{metricType}/{metricName}", metricHandler.Get)
	ts := httptest.NewServer(r)
	defer ts.Close()

	type want struct {
		code int
	}

	tests := []struct {
		name    string
		target  string
		want    want
		wantErr bool
	}{
		{
			name:   "negative get value",
			target: "/value/counter/testSetGet148",
			want: want{
				code: 404,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		resp, get := testRequest(t, ts, "GET", test.target)
		assert.Equal(t, test.want.code, resp.StatusCode, get)
	}
}

func TestMetricHandler_Update(t *testing.T) {
	r := chi.NewRouter()
	repo := repository.NewMemStorage()
	metricHandler := NewMetricHandler(repo)
	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)
	ts := httptest.NewServer(r)
	defer ts.Close()

	type want struct {
		code        int
		contentType string
	}

	tests := []struct {
		name    string
		target  string
		want    want
		wantErr bool
	}{
		{
			name:   "positive counter",
			target: "/update/counter/testSetGet20/263",
			want: want{
				code:        200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
		{
			name:   "positive gauge",
			target: "/update/gauge/Alloc/100.0",
			want: want{
				code:        200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		resp, get := testRequest(t, ts, "POST", test.target)
		assert.Equal(t, test.want.code, resp.StatusCode, get)
	}
}

func TestMetricHandler_UpdateNew(t *testing.T) {
	r := chi.NewRouter()
	repo := repository.NewMemStorage()
	metricHandler := NewMetricHandler(repo)
	r.Post("/update", metricHandler.UpdateNew)
	ts := httptest.NewServer(r)
	defer ts.Close()

	type want struct {
		code int
	}

	tests := []struct {
		name    string
		payload models.Metrics
		want    want
		wantErr bool
	}{
		{
			name: "positive gauge",
			payload: models.Metrics{
				ID:    "Alloc",
				MType: "gauge",
				Value: func() *float64 { v := 100.5; return &v }(),
			},
			want: want{
				code: 200,
			},
			wantErr: false,
		},
		{
			name: "positive counter",
			payload: models.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: func() *int64 { v := int64(5); return &v }(),
			},
			want: want{
				code: 200,
			},
			wantErr: false,
		},
		{
			name: "negative invalid type",
			payload: models.Metrics{
				ID:    "Invalid",
				MType: "unknown",
				Value: func() *float64 { v := 100.5; return &v }(),
			},
			want: want{
				code: 400,
			},
			wantErr: true,
		},
		{
			name: "positive gauge zero value",
			payload: models.Metrics{
				ID:    "ZeroGauge",
				MType: "gauge",
				Value: func() *float64 { v := 0.0; return &v }(),
			},
			want: want{
				code: 200,
			},
			wantErr: false,
		},
		{
			name: "positive counter zero value",
			payload: models.Metrics{
				ID:    "ZeroCounter",
				MType: "counter",
				Delta: func() *int64 { v := int64(0); return &v }(),
			},
			want: want{
				code: 200,
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testJSONRequest(t, ts, "POST", "/update", test.payload)
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}

func TestMetricHandler_Value(t *testing.T) {
	r := chi.NewRouter()
	repo := repository.NewMemStorage()
	metricHandler := NewMetricHandler(repo)

	// Setup: добавляем метрики
	gaugeValue := 123.45
	counterDelta := int64(10)
	_ = repo.Save(models.Metrics{
		ID:    "TestGauge",
		MType: "gauge",
		Value: &gaugeValue,
	})
	_ = repo.Save(models.Metrics{
		ID:    "TestCounter",
		MType: "counter",
		Delta: &counterDelta,
	})

	r.Post("/value", metricHandler.Value)
	ts := httptest.NewServer(r)
	defer ts.Close()

	type want struct {
		code int
	}

	tests := []struct {
		name    string
		payload models.Metrics
		want    want
		wantErr bool
	}{
		{
			name: "positive get gauge",
			payload: models.Metrics{
				ID:    "TestGauge",
				MType: "gauge",
			},
			want: want{
				code: 200,
			},
			wantErr: false,
		},
		{
			name: "positive get counter",
			payload: models.Metrics{
				ID:    "TestCounter",
				MType: "counter",
			},
			want: want{
				code: 200,
			},
			wantErr: false,
		},
		{
			name: "negative metric not found",
			payload: models.Metrics{
				ID:    "NotExists",
				MType: "gauge",
			},
			want: want{
				code: 404,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testJSONRequest(t, ts, "POST", "/value", test.payload)
			assert.Equal(t, test.want.code, resp.StatusCode)

			if !test.wantErr && resp.StatusCode == 200 {
				var result models.Metrics
				err := json.Unmarshal([]byte(body), &result)
				require.NoError(t, err)
				assert.Equal(t, test.payload.ID, result.ID)
				assert.Equal(t, test.payload.MType, result.MType)
			}
		})
	}
}
