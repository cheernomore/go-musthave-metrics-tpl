package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipWriter позволяет перехватывать запись ответа и сжимать её
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
		return w.Writer.Write(b)
	}
	// Если тип не подходит, пишем как есть в оригинальный ResponseWriter
	return w.ResponseWriter.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. РАСПАКОВКА (Input)
		// Если клиент прислал сжатые данные, распаковываем их перед обработкой
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = io.NopCloser(gz)
		}

		// 2. СЖАТИЕ (Output)
		// Если клиент поддерживает gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Создаем gzip.Writer
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		// Оборачиваем ResponseWriter, чтобы проверять Content-Type перед сжатием
		gw := &gzipWriter{
			ResponseWriter: w,
			Writer:         gz,
		}

		// Важно: заголовки проверяются внутри обработчиков,
		// поэтому нам нужно прокинуть логику сжатия через кастомный Writer.
		// Чтобы сжатие работало только для нужных типов,
		// можно добавить проверку Content-Type в методе Write (см. ниже).

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gw, r)
	})
}
