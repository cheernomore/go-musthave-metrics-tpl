package main

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

type Config struct {
	Address        string `env:"ADDRESS"`
	ReportInterval *int   `env:"REPORT_INTERVAL"`
	PollInterval   *int   `env:"POLL_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      *int   `env:"RATE_LIMIT"`
}

var flagAddressPort string
var flagReportInterval int
var flagPollInterval int
var flagKey string
var flagRateLimit int

func parseFlags() {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal("ошибка при парсинге конфига")
	}

	flag.StringVar(&flagAddressPort, "a", "localhost:8080", "port to run server")
	flag.IntVar(&flagReportInterval, "r", 10, "interval between metrics sending")
	flag.IntVar(&flagPollInterval, "p", 2, "interval between metrics pooling")
	flag.StringVar(&flagKey, "k", "", "key for signing requests")
	flag.IntVar(&flagRateLimit, "l", 1, "max concurrent outgoing requests")
	flag.Parse()

	if cfg.Address != "" {
		flagAddressPort = cfg.Address
	}

	if cfg.ReportInterval != nil {
		flagReportInterval = *cfg.ReportInterval
	}

	if cfg.PollInterval != nil {
		flagPollInterval = *cfg.PollInterval
	}

	if cfg.Key != "" {
		flagKey = cfg.Key
	}

	if cfg.RateLimit != nil {
		flagRateLimit = *cfg.RateLimit
	}
}
