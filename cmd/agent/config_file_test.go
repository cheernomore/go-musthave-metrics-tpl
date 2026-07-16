package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyAgentFile(t *testing.T) {
	content := `{
		"address": "srv:9000",
		"report_interval": "7s",
		"poll_interval": "3s",
		"crypto_key": "/pub.pem",
		"key": "sec",
		"rate_limit": 5
	}`
	path := filepath.Join(t.TempDir(), "cfg.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	fc, err := loadAgentFile(path)
	require.NoError(t, err)

	cfg := Config{Address: "localhost:8080", ReportInterval: 10, PollInterval: 2, RateLimit: 1}
	applyAgentFile(&cfg, fc)

	assert.Equal(t, "srv:9000", cfg.Address)
	assert.Equal(t, 7, cfg.ReportInterval)
	assert.Equal(t, 3, cfg.PollInterval)
	assert.Equal(t, "/pub.pem", cfg.CryptoKey)
	assert.Equal(t, "sec", cfg.Key)
	assert.Equal(t, 5, cfg.RateLimit)
}

func TestApplyAgentFile_EmptyKeepsDefaults(t *testing.T) {
	cfg := Config{Address: "localhost:8080", ReportInterval: 10, PollInterval: 2, RateLimit: 1}
	applyAgentFile(&cfg, agentFileConfig{})

	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 1, cfg.RateLimit)
}

func TestLoadAgentFile_Errors(t *testing.T) {
	_, err := loadAgentFile(filepath.Join(t.TempDir(), "nope.json"))
	assert.Error(t, err)

	bad := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte("{not json"), 0o644))
	_, err = loadAgentFile(bad)
	assert.Error(t, err)
}
