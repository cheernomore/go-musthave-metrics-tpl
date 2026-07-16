package main

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

// Config — конфигурация сервера. Каждый параметр задаётся флагом командной
// строки и может быть переопределён одноимённой переменной окружения.
type Config struct {
	// Address — адрес и порт прослушивания (флаг -a, ENV ADDRESS).
	Address string `env:"ADDRESS"`
	// StoreInterval — интервал сохранения метрик на диск в секундах (флаг -i, ENV STORE_INTERVAL).
	StoreInterval int `env:"STORE_INTERVAL"`
	// FileStoragePath — путь к файлу хранения метрик (флаг -f, ENV FILE_STORAGE_PATH).
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// Restore — восстанавливать ли метрики из файла при старте (флаг -r, ENV RESTORE).
	Restore bool `env:"RESTORE"`
	// DatabaseDSN — строка подключения к PostgreSQL (флаг -d, ENV DATABASE_DSN).
	DatabaseDSN string `env:"DATABASE_DSN"`
	// Key — ключ подписи запросов (флаг -k, ENV KEY).
	Key string `env:"KEY"`
	// AuditFile — путь к файлу аудита; пусто — аудит в файл отключён (флаг -audit-file, ENV AUDIT_FILE).
	AuditFile string `env:"AUDIT_FILE"`
	// AuditURL — URL приёмника аудита; пусто — удалённый аудит отключён (флаг -audit-url, ENV AUDIT_URL).
	AuditURL string `env:"AUDIT_URL"`
	// CryptoKey — путь к файлу с приватным RSA-ключом для расшифровки запросов
	// (флаг -crypto-key, ENV CRYPTO_KEY). Пусто — шифрование отключено.
	CryptoKey string `env:"CRYPTO_KEY"`
}

// LoadConfig разбирает флаги командной строки и переменные окружения
// и возвращает итоговую конфигурацию сервера.
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
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to RSA private key file for decrypting requests")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("ошибка при парсинге ENV: %v", err)
	}

	return cfg
}
