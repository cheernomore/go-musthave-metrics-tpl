package main

import (
	"fmt"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/handler"
	"net/http"
	"os"
)

func main() {
	fmt.Println("Server started on port 8080")
	run()
}

func run() {
	mux := http.NewServeMux()

	mux.HandleFunc("/update/{metricType}/{metricName}/{value}", handler.UpdateHandler)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Server not started", err)
		return
	}
}
