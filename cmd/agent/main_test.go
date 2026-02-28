package main

import (
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMetrics(t *testing.T) {

	testMux := http.NewServeMux()
	testMux.HandleFunc("/update/{metricType}/{metricName}/{value}", handler.UpdateHandler)
	testServer := httptest.NewServer(testMux)
	baseUrl := testServer.URL + "/update"

	type want struct {
		status      int
		contentType string
	}

	tests := []struct {
		name       string
		metricType string
		metricName string
		value      any
		want       want
		wantErr    bool
	}{
		{
			name:       "positive gauge send",
			metricType: "gauge",
			metricName: "Alloc",
			value:      1000.00,
			want: want{
				status:      200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
		{
			name:       "positive counter send",
			metricType: "counter",
			metricName: "PollCount",
			value:      int64(100),
			want: want{
				status:      200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{}
			res, err := SendMetrics(baseUrl, test.metricType, test.metricName, test.value, client)
			require.NoError(t, err)

			assert.Equal(t, test.want.status, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header)
		})
	}
}
