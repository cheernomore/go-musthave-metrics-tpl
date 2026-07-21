package main

import (
	"bytes"
	"io"
	"net/http"

	"github.com/cheernomore/go-musthave-metrics-tpl/internal/crypto"
)

// DecryptMiddleware создаёт middleware расшифровки тела запроса. Если keyPath
// пуст, возвращается «прозрачное» middleware. Иначе из файла загружается
// приватный RSA-ключ, и тела запросов, помеченных заголовком
// crypto.EncryptedHeader, расшифровываются перед дальнейшей обработкой.
func DecryptMiddleware(keyPath string) (func(http.Handler) http.Handler, error) {
	if keyPath == "" {
		return func(next http.Handler) http.Handler { return next }, nil
	}

	priv, err := crypto.LoadPrivateKey(keyPath)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(crypto.EncryptedHeader) == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "cannot read request body", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()

			plain, err := crypto.Decrypt(priv, body)
			if err != nil {
				http.Error(w, "cannot decrypt request body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plain))
			r.ContentLength = int64(len(plain))
			next.ServeHTTP(w, r)
		})
	}, nil
}
