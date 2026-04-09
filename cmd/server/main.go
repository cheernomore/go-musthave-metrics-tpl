package main

import (
	"context"
	"database/sql"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/logger"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

var db *sql.DB

func run() error {
	if err := logger.Initialize("info"); err != nil {
		return err
	}

	logger.Log.Info("--- start config loading ---")
	cfg := LoadConfig()
	logger.Log.Info("--- end config loading ---")

	// Подключение к БД опционально
	if cfg.DatabaseDSN != "" {
		logger.Log.Info("--- start connection to db")
		var err error
		db, err = sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			logger.Log.Warn("не удалось открыть соединение с БД, продолжаем работу без БД", zap.Error(err))
			db = nil
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if err = db.PingContext(ctx); err != nil {
				logger.Log.Warn("не удалось подключиться к БД, продолжаем работу без БД", zap.Error(err))
				db.Close()
				db = nil
			} else {
				defer db.Close()
				logger.Log.Info("--- end connection to db")
			}
		}
	} else {
		logger.Log.Info("DATABASE_DSN не указан, работаем без БД")
	}

	memStorage := repository.NewMemStorage()

	if cfg.Restore && cfg.FileStoragePath != "" {
		if err := memStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
			logger.Log.Warn("не удалось загрузить данные из файла", zap.Error(err))
		}
	}

	var repo repository.MetricsRepository = memStorage

	if cfg.FileStoragePath != "" && cfg.StoreInterval == 0 {
		repo = repository.NewFileBackedRepository(memStorage, cfg.FileStoragePath)
	}

	if cfg.FileStoragePath != "" && cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := memStorage.SaveToFile(cfg.FileStoragePath); err != nil {
					logger.Log.Error("ошибка сохранения в файл по тикеру", zap.Error(err))
				}
			}
		}()
	}

	metricHandler := handler.NewMetricHandler(repo)

	r := chi.NewRouter()

	r.Use(logger.RequestLogger)
	r.Use(GzipMiddleware)

	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)
	r.Post("/update", metricHandler.UpdateNew)
	r.Post("/update/", metricHandler.UpdateNew)
	r.Post("/value", metricHandler.Value)
	r.Post("/value/", metricHandler.Value)
	r.Get("/value/{metricType}/{metricName}", metricHandler.Get)
	r.Get("/ping", ping)
	r.Get("/", metricHandler.Index)

	logger.Log.Info("Running server", zap.String("address", cfg.Address))
	return http.ListenAndServe(cfg.Address, r)
}

func ping(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database connection not configured", http.StatusInternalServerError)
		return
	}

	err := db.Ping()
	if err != nil {
		http.Error(w, "db not ready", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
