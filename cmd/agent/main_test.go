package main

import (
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)

	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestSendMetrics(t *testing.T) {
	r := chi.NewRouter()
	repo := repository.NewMemStorage()
	metricHandler := handler.NewMetricHandler(repo, "", 0)
	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)

	testServer := httptest.NewServer(r)

	type want struct {
		status      int
		contentType string
	}

	tests := []struct {
		name    string
		target  string
		want    want
		wantErr bool
	}{
		{
			name:   "positive gauge send",
			target: "/update/gauge/Alloc/234.234",
			want: want{
				status:      200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
		{
			name:   "positive counter send",
			target: "/update/counter/PollCount/1000",
			want: want{
				status:      200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, get := testRequest(t, testServer, "POST", test.target)
			defer resp.Body.Close()
			assert.Equal(t, test.want.status, resp.StatusCode, get)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"), get)
		})
	}
}
