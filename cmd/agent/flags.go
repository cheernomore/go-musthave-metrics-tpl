package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/config"
)

// Значения интервалов агента по умолчанию (в секундах).
const (
	defaultPollInterval   = 2
	defaultReportInterval = 10
)

// Config — конфигурация агента. Каждый параметр задаётся флагом командной
// строки и может быть переопределён одноимённой переменной окружения. Кроме
// того, значения можно задать через JSON-файл (флаг -c/-config или ENV CONFIG),
// который имеет наименьший приоритет.
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
	// CryptoKey — путь к файлу с публичным RSA-ключом для шифрования запросов
	// (флаг -crypto-key, ENV CRYPTO_KEY). Пусто — шифрование отключено.
	CryptoKey string `env:"CRYPTO_KEY"`
	// GRPCAddress — адрес gRPC-сервера (флаг -g, ENV GRPC_ADDRESS). Если задан,
	// метрики отправляются по gRPC вместо HTTP.
	GRPCAddress string `env:"GRPC_ADDRESS"`
}

// agentFileConfig — представление JSON-файла конфигурации агента. Поля
// объявлены указателями, чтобы отличать заданное значение от отсутствующего.
type agentFileConfig struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
	GRPCAddress    *string `json:"grpc_address"`
	Key            *string `json:"key"`
	RateLimit      *int    `json:"rate_limit"`
}

// loadAgentFile читает и разбирает JSON-файл конфигурации агента.
func loadAgentFile(path string) (agentFileConfig, error) {
	var fc agentFileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return fc, err
	}
	if err := json.Unmarshal(data, &fc); err != nil {
		return fc, err
	}
	return fc, nil
}

// applyAgentFile накладывает заданные в файле значения на cfg.
func applyAgentFile(cfg *Config, fc agentFileConfig) {
	if fc.Address != nil {
		cfg.Address = *fc.Address
	}
	if fc.ReportInterval != nil {
		if sec, err := config.Seconds(*fc.ReportInterval); err == nil {
			cfg.ReportInterval = sec
		}
	}
	if fc.PollInterval != nil {
		if sec, err := config.Seconds(*fc.PollInterval); err == nil {
			cfg.PollInterval = sec
		}
	}
	if fc.CryptoKey != nil {
		cfg.CryptoKey = *fc.CryptoKey
	}
	if fc.GRPCAddress != nil {
		cfg.GRPCAddress = *fc.GRPCAddress
	}
	if fc.Key != nil {
		cfg.Key = *fc.Key
	}
	if fc.RateLimit != nil {
		cfg.RateLimit = *fc.RateLimit
	}
}

// parseFlags собирает конфигурацию агента. Приоритет (от высшего к низшему):
// переменные окружения, флаги командной строки, файл конфигурации, значения
// по умолчанию.
func parseFlags() Config {
	def := Config{
		Address:        "localhost:8080",
		ReportInterval: defaultReportInterval,
		PollInterval:   defaultPollInterval,
		RateLimit:      1,
	}

	if path := config.Path(os.Args[1:], os.Getenv("CONFIG")); path != "" {
		if fc, err := loadAgentFile(path); err != nil {
			log.Printf("не удалось прочитать файл конфигурации %s: %v", path, err)
		} else {
			applyAgentFile(&def, fc)
		}
	}

	var cfg Config
	var configPath string
	flag.StringVar(&configPath, "c", "", "path to JSON config file")
	flag.StringVar(&configPath, "config", "", "path to JSON config file")

	flag.StringVar(&cfg.Address, "a", def.Address, "port to run server")
	flag.IntVar(&cfg.ReportInterval, "r", def.ReportInterval, "interval between metrics sending")
	flag.IntVar(&cfg.PollInterval, "p", def.PollInterval, "interval between metrics pooling")
	flag.StringVar(&cfg.Key, "k", def.Key, "key for signing requests")
	flag.IntVar(&cfg.RateLimit, "l", def.RateLimit, "max concurrent outgoing requests")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", def.CryptoKey, "path to RSA public key file for encrypting requests")
	flag.StringVar(&cfg.GRPCAddress, "g", def.GRPCAddress, "address of gRPC server (if set, metrics are sent via gRPC)")
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
