package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

func calculateHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

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
	body   *bytes.Buffer
	status int
	key    string
}

func newResponseWriterWithHash(w http.ResponseWriter, key string) *responseWriterWithHash {
	return &responseWriterWithHash{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		status:         http.StatusOK,
		key:            key,
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

	rw.ResponseWriter.WriteHeader(rw.status)
	rw.ResponseWriter.Write(rw.body.Bytes())
}

func HashResponseMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			wrapped := newResponseWriterWithHash(w, key)

			next.ServeHTTP(wrapped, r)

			wrapped.flush()
		})
	}
}
