package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyServerFile(t *testing.T) {
	content := `{
		"address": "1.2.3.4:9000",
		"restore": false,
		"store_interval": "5s",
		"store_file": "/data/m.db",
		"database_dsn": "postgres://x",
		"crypto_key": "/k.pem",
		"key": "secret",
		"audit_file": "/a.log",
		"audit_url": "http://a"
	}`
	path := filepath.Join(t.TempDir(), "cfg.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	fc, err := loadServerFile(path)
	require.NoError(t, err)

	cfg := Config{Address: "localhost:8080", StoreInterval: 300, FileStoragePath: "/tmp/x", Restore: true}
	applyServerFile(&cfg, fc)

	assert.Equal(t, "1.2.3.4:9000", cfg.Address)
	assert.False(t, cfg.Restore)
	assert.Equal(t, 5, cfg.StoreInterval)
	assert.Equal(t, "/data/m.db", cfg.FileStoragePath)
	assert.Equal(t, "postgres://x", cfg.DatabaseDSN)
	assert.Equal(t, "/k.pem", cfg.CryptoKey)
	assert.Equal(t, "secret", cfg.Key)
	assert.Equal(t, "/a.log", cfg.AuditFile)
	assert.Equal(t, "http://a", cfg.AuditURL)
}

func TestApplyServerFile_EmptyKeepsDefaults(t *testing.T) {
	cfg := Config{Address: "localhost:8080", StoreInterval: 300, Restore: true}
	applyServerFile(&cfg, serverFileConfig{})

	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.True(t, cfg.Restore)
}

func TestLoadServerFile_Errors(t *testing.T) {
	_, err := loadServerFile(filepath.Join(t.TempDir(), "nope.json"))
	assert.Error(t, err)

	bad := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte("{not json"), 0o644))
	_, err = loadServerFile(bad)
	assert.Error(t, err)
}
