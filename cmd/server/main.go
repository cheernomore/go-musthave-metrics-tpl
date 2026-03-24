package main

import (
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/logger"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	parseFlags()
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := logger.Initialize("info"); err != nil {
		return err
	}
	r := chi.NewRouter()
	repo := repository.NewMemStorage()
	metricHandler := handler.NewMetricHandler(repo)

	r.Use(logger.RequestLogger)
	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)
	r.Post("/update", metricHandler.UpdateNew)
	r.Get("/value", metricHandler.Value)
	r.Get("/value/{metricType}/{metricName}", metricHandler.Get)
	r.Get("/", metricHandler.Index)

	logger.Log.Info("Running server", zap.String("address", flagAddressPort))
	return http.ListenAndServe(flagAddressPort, r)
}
