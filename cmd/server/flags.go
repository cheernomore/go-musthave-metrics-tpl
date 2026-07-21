package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/config"
)

// Config — конфигурация сервера. Каждый параметр задаётся флагом командной
// строки и может быть переопределён одноимённой переменной окружения. Кроме
// того, значения можно задать через JSON-файл (флаг -c/-config или ENV CONFIG),
// который имеет наименьший приоритет.
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
	// TrustedSubnet — доверенная подсеть в формате CIDR (флаг -t,
	// ENV TRUSTED_SUBNET). Пусто — метрики принимаются без ограничений.
	TrustedSubnet string `env:"TRUSTED_SUBNET"`
	// GRPCAddress — адрес gRPC-сервера (флаг -g, ENV GRPC_ADDRESS).
	// Пусто — gRPC-сервер не запускается.
	GRPCAddress string `env:"GRPC_ADDRESS"`
}

// serverFileConfig — представление JSON-файла конфигурации сервера. Поля
// объявлены указателями, чтобы отличать заданное значение от отсутствующего.
type serverFileConfig struct {
	Address       *string `json:"address"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"`
	StoreFile     *string `json:"store_file"`
	DatabaseDSN   *string `json:"database_dsn"`
	CryptoKey     *string `json:"crypto_key"`
	TrustedSubnet *string `json:"trusted_subnet"`
	GRPCAddress   *string `json:"grpc_address"`
	Key           *string `json:"key"`
	AuditFile     *string `json:"audit_file"`
	AuditURL      *string `json:"audit_url"`
}

// loadServerFile читает и разбирает JSON-файл конфигурации сервера.
func loadServerFile(path string) (serverFileConfig, error) {
	var fc serverFileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return fc, err
	}
	if err := json.Unmarshal(data, &fc); err != nil {
		return fc, err
	}
	return fc, nil
}

// applyServerFile накладывает заданные в файле значения на cfg.
func applyServerFile(cfg *Config, fc serverFileConfig) {
	if fc.Address != nil {
		cfg.Address = *fc.Address
	}
	if fc.Restore != nil {
		cfg.Restore = *fc.Restore
	}
	if fc.StoreInterval != nil {
		if sec, err := config.Seconds(*fc.StoreInterval); err == nil {
			cfg.StoreInterval = sec
		}
	}
	if fc.StoreFile != nil {
		cfg.FileStoragePath = *fc.StoreFile
	}
	if fc.DatabaseDSN != nil {
		cfg.DatabaseDSN = *fc.DatabaseDSN
	}
	if fc.CryptoKey != nil {
		cfg.CryptoKey = *fc.CryptoKey
	}
	if fc.TrustedSubnet != nil {
		cfg.TrustedSubnet = *fc.TrustedSubnet
	}
	if fc.GRPCAddress != nil {
		cfg.GRPCAddress = *fc.GRPCAddress
	}
	if fc.Key != nil {
		cfg.Key = *fc.Key
	}
	if fc.AuditFile != nil {
		cfg.AuditFile = *fc.AuditFile
	}
	if fc.AuditURL != nil {
		cfg.AuditURL = *fc.AuditURL
	}
}

// LoadConfig собирает конфигурацию сервера. Приоритет (от высшего к низшему):
// переменные окружения, флаги командной строки, файл конфигурации, значения
// по умолчанию.
func LoadConfig() Config {
	// Значения по умолчанию, поверх которых накладывается файл конфигурации.
	def := Config{
		Address:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
	}

	if path := config.Path(os.Args[1:], os.Getenv("CONFIG")); path != "" {
		if fc, err := loadServerFile(path); err != nil {
			log.Printf("не удалось прочитать файл конфигурации %s: %v", path, err)
		} else {
			applyServerFile(&def, fc)
		}
	}

	var cfg Config
	// Флаг конфигурации регистрируется, чтобы flag.Parse его распознавал;
	// путь уже определён выше.
	var configPath string
	flag.StringVar(&configPath, "c", "", "path to JSON config file")
	flag.StringVar(&configPath, "config", "", "path to JSON config file")

	flag.StringVar(&cfg.Address, "a", def.Address, "address and port to run server")
	flag.IntVar(&cfg.StoreInterval, "i", def.StoreInterval, "interval in seconds to save metrics to disk")
	flag.StringVar(&cfg.FileStoragePath, "f", def.FileStoragePath, "file path to store metrics")
	flag.BoolVar(&cfg.Restore, "r", def.Restore, "restore metrics from file on start")
	flag.StringVar(&cfg.DatabaseDSN, "d", def.DatabaseDSN, "connect to db")
	flag.StringVar(&cfg.Key, "k", def.Key, "key for signing requests")
	flag.StringVar(&cfg.AuditFile, "audit-file", def.AuditFile, "path to audit log file (audit disabled if empty)")
	flag.StringVar(&cfg.AuditURL, "audit-url", def.AuditURL, "url to send audit events via POST (audit disabled if empty)")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", def.CryptoKey, "path to RSA private key file for decrypting requests")
	flag.StringVar(&cfg.TrustedSubnet, "t", def.TrustedSubnet, "trusted subnet in CIDR notation (empty — no restrictions)")
	flag.StringVar(&cfg.GRPCAddress, "g", def.GRPCAddress, "address of gRPC server (empty — gRPC disabled)")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("ошибка при парсинге ENV: %v", err)
	}

	return cfg
}
