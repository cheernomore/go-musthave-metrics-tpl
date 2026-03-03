package main

import (
	"fmt"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
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
	r.Post("/update/{metricType}/{metricName}/{value}", handler.Update)
	r.Get("/value/{metricType}/{metricName}", handler.Get)
	r.Get("/", handler.Index)

	fmt.Println("running server on port: ", flagAddressPort)
	return http.ListenAndServe(flagAddressPort, r)
}
