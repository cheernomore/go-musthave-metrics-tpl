package handler

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateHandler(t *testing.T) {
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
			target: "/update/counter/metricName/100",
			want: want{
				code:        200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
		{
			name:   "positive gauge",
			target: "/update/gauge/metricName/100.0",
			want: want{
				code:        200,
				contentType: "text/plain; charset=utf-8",
			},
			wantErr: false,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/update/{metricType}/{metricName}/{value}", UpdateHandler)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, test.target, nil)
			w := httptest.NewRecorder()
			req.Header.Set("Content-Type", "text/plain")
			mux.ServeHTTP(w, req)
			res := w.Result()

			body, _ := io.ReadAll(w.Body)

			assert.Equal(t, test.want.code, res.StatusCode, string(body))
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
