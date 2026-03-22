package main

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

type Config struct {
	Address string `env:"ADDRESS"`
}

var flagAddressPort string

func parseFlags() {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal("ошибка при парсинге ENV")
	}

	flag.StringVar(&flagAddressPort, "a", "localhost:8080", "port to run server")
	flag.Parse()

	if cfg.Address != "" {
		flagAddressPort = cfg.Address
	}
}
