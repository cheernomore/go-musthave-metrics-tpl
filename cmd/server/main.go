package main

import (
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/logger"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := logger.Initialize("info"); err != nil {
		return err
	}

	cfg := LoadConfig()
	repo := repository.NewMemStorage()

	metricHandler := handler.NewMetricHandler(repo, cfg.FileStoragePath, cfg.StoreInterval)

	if cfg.Restore && cfg.FileStoragePath != "" {
		if err := repo.LoadFromFile(cfg.FileStoragePath); err != nil {
			logger.Log.Warn("не удалось загрузить данные из файла", zap.Error(err))
		}
	}

	if cfg.FileStoragePath != "" && cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := repo.SaveToFile(cfg.FileStoragePath); err != nil {
					logger.Log.Error("ошибка сохранения в файл по тикеру", zap.Error(err))
				}
			}
		}()
	}

	r := chi.NewRouter()

	r.Use(logger.RequestLogger)
	r.Use(GzipMiddleware)

	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)
	r.Post("/update", metricHandler.UpdateNew)
	r.Post("/update/", metricHandler.UpdateNew)
	r.Post("/value", metricHandler.Value)
	r.Post("/value/", metricHandler.Value)
	r.Get("/value/{metricType}/{metricName}", metricHandler.Get)
	r.Get("/", metricHandler.Index)

	logger.Log.Info("Running server", zap.String("address", cfg.Address))
	return http.ListenAndServe(cfg.Address, r)
}
