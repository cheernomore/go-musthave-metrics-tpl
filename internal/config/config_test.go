package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  string
		want string
	}{
		{"из env", nil, "env.json", "env.json"},
		{"флаг -c со значением", []string{"-c", "flag.json"}, "", "flag.json"},
		{"флаг -config со значением", []string{"-config", "flag.json"}, "", "flag.json"},
		{"флаг -c=value", []string{"-c=flag.json"}, "", "flag.json"},
		{"флаг --config=value", []string{"--config=flag.json"}, "", "flag.json"},
		{"флаг перекрывает env", []string{"-c", "flag.json"}, "env.json", "flag.json"},
		{"нет ни флага, ни env", []string{"-a", "localhost"}, "", ""},
		{"crypto-key не путается с -c", []string{"-crypto-key", "k.pem"}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Path(tt.args, tt.env))
		})
	}
}

func TestSeconds(t *testing.T) {
	sec, err := Seconds("1s")
	require.NoError(t, err)
	assert.Equal(t, 1, sec)

	sec, err = Seconds("10s")
	require.NoError(t, err)
	assert.Equal(t, 10, sec)

	_, err = Seconds("не длительность")
	assert.Error(t, err)
}
