package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(data)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestGzipMiddleware_DecompressesRequest(t *testing.T) {
	var received string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		w.WriteHeader(http.StatusOK)
	})

	body := gzipBytes(t, []byte("hello gzip"))
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello gzip", received)
}

func TestGzipMiddleware_CompressesResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gz, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer gz.Close()
	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(decompressed))
}

func TestGzipMiddleware_PassthroughWithoutGzip(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	GzipMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, "plain", rec.Body.String())
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
}

func TestCalculateHash(t *testing.T) {
	a := calculateHash([]byte("data"), "key")
	b := calculateHash([]byte("data"), "key")
	c := calculateHash([]byte("data"), "other")

	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
	assert.Len(t, a, 64)
}

func TestHashValidationMiddleware(t *testing.T) {
	const key = "secret"
	body := []byte(`{"id":"Alloc"}`)
	validHash := calculateHash(body, key)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid hash passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		req.Header.Set("HashSHA256", validHash)
		rec := httptest.NewRecorder()
		HashValidationMiddleware(key)(next).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid hash rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		req.Header.Set("HashSHA256", "deadbeef")
		rec := httptest.NewRecorder()
		HashValidationMiddleware(key)(next).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("empty key disables validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		HashValidationMiddleware("")(next).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHashResponseMiddleware(t *testing.T) {
	const key = "secret"
	payload := []byte("response-body")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	HashResponseMiddleware(key)(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, payload, rec.Body.Bytes())
	assert.Equal(t, calculateHash(payload, key), rec.Header().Get("HashSHA256"))
}
