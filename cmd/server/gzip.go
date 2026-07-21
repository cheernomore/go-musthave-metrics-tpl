package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// shouldCompress сообщает, подлежит ли ответ сжатию по его Content-Type.
func (w gzipWriter) shouldCompress() bool {
	contentType := w.Header().Get("Content-Type")
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html")
}

// WriteHeader выставляет Content-Encoding до отправки статус-кода, иначе
// заголовок, добавленный позже в Write, уже не попал бы в ответ (гонка
// заголовков при явном вызове WriteHeader обработчиком).
func (w gzipWriter) WriteHeader(statusCode int) {
	if w.shouldCompress() {
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w gzipWriter) Write(b []byte) (int, error) {
	if w.shouldCompress() {
		w.Header().Set("Content-Encoding", "gzip")
		return w.Writer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// GzipMiddleware распаковывает входящие запросы с Content-Encoding: gzip
// и сжимает ответ, если клиент поддерживает gzip (Accept-Encoding) и тип
// содержимого подходит для сжатия (JSON или HTML).
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. РАСПАКОВКА (Входящий запрос)
		// Проверяем наличие заголовка gzip
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				// Если заголовок есть, но данные битые — это ошибка
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = io.NopCloser(gz)
		}

		// 2. СЖАТИЕ (Исходящий ответ)
		// Проверяем, поддерживает ли клиент gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		// Передаем кастомный ResponseWriter
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}
