package main

import (
	"fmt"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"github.com/go-chi/chi"
	"net/http"
	"os"
)

func main() {
	fmt.Println("Server started on port 8080") //	like a log)))
	run()
}

func run() {
	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{value}", handler.Update)
	r.Get("/value/{metricType}/{metricName}", handler.Get)

	r.Get("/", handler.Index)

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Server not started", err)
		return
	}
}
