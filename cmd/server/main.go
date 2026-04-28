package main

import (
	"context"
	"database/sql"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/logger"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Server struct {
	db *sql.DB
}

func NewServer() *Server {
	return &Server{}
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	srv := NewServer()
	if err := logger.Initialize("info"); err != nil {
		return err
	}

	logger.Log.Info("--- start config loading ---")
	cfg := LoadConfig()
	logger.Log.Info("--- end config loading ---")

	var repo repository.MetricsRepository

	if cfg.DatabaseDSN != "" {
		logger.Log.Info("--- start connection to db")
		var err error
		srv.db, err = sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			logger.Log.Warn("не удалось открыть соединение с БД, fallback на файловое хранилище", zap.Error(err))
			srv.db = nil
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if err = srv.db.PingContext(ctx); err != nil {
				logger.Log.Warn("не удалось подключиться к БД, fallback на файловое хранилище", zap.Error(err))
				srv.db.Close()
				srv.db = nil
			} else {
				defer srv.db.Close()
				logger.Log.Info("--- успешное подключение к БД")

				if err := runMigrations(srv.db); err != nil {
					logger.Log.Warn("не удалось применить миграции, fallback на файловое хранилище", zap.Error(err))
					srv.db.Close()
					srv.db = nil
				} else {
					logger.Log.Info("используем PostgreSQL для хранения метрик")
					repo = repository.NewPostgresRepository(srv.db)
				}
			}
		}
	} else {
		logger.Log.Info("DATABASE_DSN не указан")
	}

	if repo == nil {
		memStorage := repository.NewMemStorage()

		if cfg.Restore && cfg.FileStoragePath != "" {
			if err := memStorage.LoadFromFile(cfg.FileStoragePath); err != nil {
				logger.Log.Warn("не удалось загрузить данные из файла", zap.Error(err))
			} else {
				logger.Log.Info("данные восстановлены из файла")
			}
		}

		repo = memStorage

		if cfg.FileStoragePath != "" && cfg.StoreInterval == 0 {
			logger.Log.Info("используем синхронное сохранение в файл")
			repo = repository.NewFileBackedRepository(memStorage, cfg.FileStoragePath)
		}

		if cfg.FileStoragePath != "" && cfg.StoreInterval > 0 {
			logger.Log.Info("используем периодическое сохранение в файл", zap.Int("interval", cfg.StoreInterval))
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

		if cfg.FileStoragePath == "" {
			logger.Log.Info("используем только память для хранения метрик")
		}
	}

	metricHandler := handler.NewMetricHandler(repo)

	r := chi.NewRouter()

	r.Use(logger.RequestLogger)
	r.Use(GzipMiddleware)
	r.Use(HashValidationMiddleware(cfg.Key))
	r.Use(HashResponseMiddleware(cfg.Key))

	r.Post("/updates/", metricHandler.Updates)
	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)
	r.Post("/update", metricHandler.UpdateNew)
	r.Post("/update/", metricHandler.UpdateNew)
	r.Post("/value", metricHandler.Value)
	r.Post("/value/", metricHandler.Value)
	r.Get("/value/{metricType}/{metricName}", metricHandler.Get)
	r.Get("/ping", srv.ping)
	r.Get("/", metricHandler.Index)

	logger.Log.Info("Running server", zap.String("address", cfg.Address))
	return http.ListenAndServe(cfg.Address, r)
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "database connection not configured", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	err := s.db.PingContext(ctx)
	if err != nil {
		http.Error(w, "db not ready", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
