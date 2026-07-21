package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildBatch(t *testing.T) {
	metrics := []Metric{
		{"Alloc", uint64(10), "gauge"},
		{"PollCount", int64(3), "counter"},
		{"Bad", "не число", "gauge"}, // не конвертируется — пропускается
	}

	batch := buildBatch(metrics)

	require.Len(t, batch, 2)
	assert.Equal(t, "Alloc", batch[0].ID)
	assert.Equal(t, "PollCount", batch[1].ID)
}

func TestBuildBatch_Empty(t *testing.T) {
	assert.Empty(t, buildBatch(nil))
}
