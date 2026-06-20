package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_NormalizeIntervals(t *testing.T) {
	tests := []struct {
		name       string
		in         Config
		wantPoll   int
		wantReport int
	}{
		{
			name:       "корректные значения сохраняются",
			in:         Config{PollInterval: 3, ReportInterval: 15},
			wantPoll:   3,
			wantReport: 15,
		},
		{
			name:       "нулевые заменяются на дефолтные",
			in:         Config{PollInterval: 0, ReportInterval: 0},
			wantPoll:   defaultPollInterval,
			wantReport: defaultReportInterval,
		},
		{
			name:       "отрицательные заменяются на дефолтные",
			in:         Config{PollInterval: -5, ReportInterval: -1},
			wantPoll:   defaultPollInterval,
			wantReport: defaultReportInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.in
			cfg.normalizeIntervals()
			assert.Equal(t, tt.wantPoll, cfg.PollInterval)
			assert.Equal(t, tt.wantReport, cfg.ReportInterval)
			assert.Positive(t, cfg.PollInterval)
			assert.Positive(t, cfg.ReportInterval)
		})
	}
}
