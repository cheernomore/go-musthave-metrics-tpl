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

// TestGzipMiddleware_HeaderBeforeStatus проверяет, что Content-Encoding
// выставляется до отправки статус-кода. С реальным http.Server заголовки
// «застывают» на WriteHeader, поэтому без переопределения WriteHeader этот
// тест не прошёл бы (httptest.NewRecorder гонку не воспроизводит).
func TestGzipMiddleware_HeaderBeforeStatus(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // явный WriteHeader до Write
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	ts := httptest.NewServer(GzipMiddleware(h))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)
	// Явный Accept-Encoding отключает автораспаковку в Transport.
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	gz, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	defer gz.Close()
	body, err := io.ReadAll(gz)
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(body))
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

func TestGzipMiddleware_BadGzipBody(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte("это не gzip")))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHashResponseMiddleware_EmptyKeyPassthrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("body"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	HashResponseMiddleware("")(next).ServeHTTP(rec, req)

	assert.Equal(t, "body", rec.Body.String())
	assert.Empty(t, rec.Header().Get("HashSHA256"))
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
