package main

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

// Значения интервалов агента по умолчанию (в секундах).
const (
	defaultPollInterval   = 2
	defaultReportInterval = 10
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
	flag.IntVar(&cfg.ReportInterval, "r", defaultReportInterval, "interval between metrics sending")
	flag.IntVar(&cfg.PollInterval, "p", defaultPollInterval, "interval between metrics pooling")
	flag.StringVar(&cfg.Key, "k", "", "key for signing requests")
	flag.IntVar(&cfg.RateLimit, "l", 1, "max concurrent outgoing requests")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatal("ошибка при парсинге конфига")
	}

	cfg.normalizeIntervals()

	return cfg
}

// normalizeIntervals заменяет некорректные (неположительные) интервалы опроса
// и отправки значениями по умолчанию. Это защищает от паники time.NewTicker
// при передаче нулей или отрицательных значений через флаги или окружение.
func (c *Config) normalizeIntervals() {
	if c.PollInterval <= 0 {
		log.Printf("некорректный интервал опроса (%d), используется значение по умолчанию: %d",
			c.PollInterval, defaultPollInterval)
		c.PollInterval = defaultPollInterval
	}
	if c.ReportInterval <= 0 {
		log.Printf("некорректный интервал отправки (%d), используется значение по умолчанию: %d",
			c.ReportInterval, defaultReportInterval)
		c.ReportInterval = defaultReportInterval
	}
}
