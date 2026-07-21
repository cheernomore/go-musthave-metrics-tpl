package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cheernomore/go-musthave-metrics-tpl/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePrivateKey(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "private.pem")
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o644))
	return path, key
}

func echoBody(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	_, _ = w.Write(body)
}

func TestDecryptMiddleware_DecryptsBody(t *testing.T) {
	keyPath, key := writePrivateKey(t)

	mw, err := DecryptMiddleware(keyPath)
	require.NoError(t, err)

	payload := []byte("важное сообщение с метриками")
	enc, err := crypto.Encrypt(&key.PublicKey, payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(enc))
	req.Header.Set(crypto.EncryptedHeader, "1")
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(echoBody)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, payload, rec.Body.Bytes())
}

func TestDecryptMiddleware_PassthroughWithoutHeader(t *testing.T) {
	keyPath, _ := writePrivateKey(t)

	mw, err := DecryptMiddleware(keyPath)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte("plain")))
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(echoBody)).ServeHTTP(rec, req)

	assert.Equal(t, "plain", rec.Body.String())
}

func TestDecryptMiddleware_NoKeyIsTransparent(t *testing.T) {
	mw, err := DecryptMiddleware("")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte("plain")))
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(echoBody)).ServeHTTP(rec, req)

	assert.Equal(t, "plain", rec.Body.String())
}

func TestDecryptMiddleware_BadKeyPath(t *testing.T) {
	_, err := DecryptMiddleware(filepath.Join(t.TempDir(), "missing.pem"))
	assert.Error(t, err)
}

func TestDecryptMiddleware_InvalidCiphertext(t *testing.T) {
	keyPath, _ := writePrivateKey(t)
	mw, err := DecryptMiddleware(keyPath)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte("garbage")))
	req.Header.Set(crypto.EncryptedHeader, "1")
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(echoBody)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
