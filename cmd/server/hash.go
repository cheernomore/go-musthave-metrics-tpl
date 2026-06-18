package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

func calculateHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HashValidationMiddleware проверяет подпись входящего запроса. Если key пуст,
// проверка отключена. Иначе при наличии заголовка HashSHA256 сравнивает его
// с HMAC-SHA256 тела запроса и отвечает 400 при несовпадении.
func HashValidationMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			receivedHash := r.Header.Get("HashSHA256")

			if receivedHash != "" {

				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "cannot read request body", http.StatusBadRequest)
					return
				}
				defer r.Body.Close()

				r.Body = io.NopCloser(bytes.NewBuffer(body))

				expectedHash := calculateHash(body, key)

				if receivedHash != expectedHash {
					http.Error(w, "hash mismatch", http.StatusBadRequest)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriterWithHash struct {
	http.ResponseWriter
	body    *bytes.Buffer
	status  int
	key     string
	request *http.Request
}

func newResponseWriterWithHash(w http.ResponseWriter, key string, r *http.Request) *responseWriterWithHash {
	return &responseWriterWithHash{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		status:         http.StatusOK,
		key:            key,
		request:        r,
	}
}

func (rw *responseWriterWithHash) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return len(b), nil
}

func (rw *responseWriterWithHash) WriteHeader(statusCode int) {
	rw.status = statusCode
}

func (rw *responseWriterWithHash) flush() {
	if rw.key != "" && rw.body.Len() > 0 {
		hash := calculateHash(rw.body.Bytes(), rw.key)
		rw.ResponseWriter.Header().Set("HashSHA256", hash)
	}

	contentType := rw.Header().Get("Content-Type")
	acceptsGzip := strings.Contains(rw.request.Header.Get("Accept-Encoding"), "gzip")
	if acceptsGzip && (strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html")) {
		rw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	}

	rw.ResponseWriter.WriteHeader(rw.status)
	rw.ResponseWriter.Write(rw.body.Bytes())
}

// HashResponseMiddleware подписывает ответ сервера: при непустом key
// добавляет в ответ заголовок HashSHA256 с HMAC-SHA256 тела ответа.
func HashResponseMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			wrapped := newResponseWriterWithHash(w, key, r)

			next.ServeHTTP(wrapped, r)

			wrapped.flush()
		})
	}
}
