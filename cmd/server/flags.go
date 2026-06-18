package main

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

type Config struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
}

func LoadConfig() Config {
	var cfg Config

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "interval in seconds to save metrics to disk")
	flag.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "file path to store metrics")
	flag.BoolVar(&cfg.Restore, "r", true, "restore metrics from file on start")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "connect to db")
	flag.StringVar(&cfg.Key, "k", "", "key for signing requests")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "path to audit log file (audit disabled if empty)")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "url to send audit events via POST (audit disabled if empty)")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("ошибка при парсинге ENV: %v", err)
	}

	return cfg
}
