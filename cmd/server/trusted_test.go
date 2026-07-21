package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestTrustedSubnetMiddleware_EmptySubnetAllowsAll(t *testing.T) {
	mw, err := TrustedSubnetMiddleware("")
	require.NoError(t, err)

	// Без заголовка X-Real-IP запрос всё равно проходит.
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	rec := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestTrustedSubnetMiddleware_Access(t *testing.T) {
	tests := []struct {
		name   string
		subnet string
		realIP string
		want   int
	}{
		{"IP в подсети", "192.168.1.0/24", "192.168.1.42", http.StatusOK},
		{"граница подсети", "192.168.1.0/24", "192.168.1.255", http.StatusOK},
		{"IP вне подсети", "192.168.1.0/24", "10.0.0.1", http.StatusForbidden},
		{"пустой заголовок", "192.168.1.0/24", "", http.StatusForbidden},
		{"некорректный IP", "192.168.1.0/24", "не-ip", http.StatusForbidden},
		{"пробелы вокруг IP", "192.168.1.0/24", "  192.168.1.7  ", http.StatusOK},
		{"loopback в /8", "127.0.0.0/8", "127.0.0.1", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, err := TrustedSubnetMiddleware(tt.subnet)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
			if tt.realIP != "" {
				req.Header.Set(realIPHeader, tt.realIP)
			}
			rec := httptest.NewRecorder()
			mw(okHandler()).ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

func TestTrustedSubnetMiddleware_InvalidCIDR(t *testing.T) {
	_, err := TrustedSubnetMiddleware("не CIDR")
	assert.Error(t, err)

	_, err = TrustedSubnetMiddleware("192.168.1.1")
	assert.Error(t, err)
}
