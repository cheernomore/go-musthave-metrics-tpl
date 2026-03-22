package handler

import (
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
	req.Header.Set("Content-Type", "text/plain")
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

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
		defer resp.Body.Close()
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
		defer resp.Body.Close()
		assert.Equal(t, test.want.code, resp.StatusCode, get)
	}
}
