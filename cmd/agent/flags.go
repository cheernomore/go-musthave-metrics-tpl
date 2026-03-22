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
}

var flagAddressPort string
var flagReportInterval int
var flagPollInterval int

func parseFlags() {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal("ошибка при парсинге конфига")
	}

	flag.StringVar(&flagAddressPort, "a", "localhost:8080", "port to run server")
	flag.IntVar(&flagReportInterval, "r", 10, "interval between metrics sending")
	flag.IntVar(&flagPollInterval, "p", 2, "interval between metrics pooling")
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
}
