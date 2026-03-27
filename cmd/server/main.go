package main

import (
	"fmt"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/go-chi/chi"
	"net/http"
)

func main() {
	parseFlags()
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	r := chi.NewRouter()
	repo := repository.NewMemStorage()
	metricHandler := handler.NewMetricHandler(repo)

	r.Post("/update/{metricType}/{metricName}/{value}", metricHandler.Update)
	r.Get("/value/{metricType}/{metricName}", metricHandler.Get)
	r.Get("/", metricHandler.Index)

	fmt.Println("running server on port: ", flagAddressPort)
	return http.ListenAndServe(flagAddressPort, r)
}
