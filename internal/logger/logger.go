// Package logger предоставляет общий структурированный логгер на базе zap
// и HTTP-middleware для логирования запросов.
package logger

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

// Log — глобальный логгер пакета. До вызова Initialize это no-op логгер.
var Log = zap.NewNop()

// Initialize настраивает глобальный логгер Log на заданный уровень
// (например, "info", "debug"). Возвращает ошибку при некорректном уровне.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl
	return nil
}

// RequestLogger — middleware, логирующее каждый HTTP-запрос: URI, метод,
// код ответа, размер тела и длительность обработки.
func RequestLogger(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.RequestURI
		method := r.Method

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)
		Log.Info("request processing",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.Int("status", responseData.status),
			zap.Duration("duration", duration),
			zap.Int("size", responseData.size),
		)
	}
	return http.HandlerFunc(logFn)
}

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}
