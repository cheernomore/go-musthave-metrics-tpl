package main

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

// Config — конфигурация агента. Каждый параметр задаётся флагом командной
// строки и может быть переопределён одноимённой переменной окружения.
type Config struct {
	// Address — адрес сервера сбора метрик (флаг -a, ENV ADDRESS).
	Address string `env:"ADDRESS"`
	// ReportInterval — интервал отправки метрик в секундах (флаг -r, ENV REPORT_INTERVAL).
	ReportInterval int `env:"REPORT_INTERVAL"`
	// PollInterval — интервал сбора метрик в секундах (флаг -p, ENV POLL_INTERVAL).
	PollInterval int `env:"POLL_INTERVAL"`
	// Key — ключ подписи запросов (флаг -k, ENV KEY).
	Key string `env:"KEY"`
	// RateLimit — максимум одновременных исходящих запросов (флаг -l, ENV RATE_LIMIT).
	RateLimit int `env:"RATE_LIMIT"`
}

func parseFlags() Config {
	var cfg Config

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "port to run server")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "interval between metrics sending")
	flag.IntVar(&cfg.PollInterval, "p", 2, "interval between metrics pooling")
	flag.StringVar(&cfg.Key, "k", "", "key for signing requests")
	flag.IntVar(&cfg.RateLimit, "l", 1, "max concurrent outgoing requests")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatal("ошибка при парсинге конфига")
	}

	return cfg
}
