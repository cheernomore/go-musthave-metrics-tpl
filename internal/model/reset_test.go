package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_Reset(t *testing.T) {
	delta := int64(5)
	value := 3.14
	m := &Metrics{
		ID:    "Alloc",
		MType: Gauge,
		Delta: &delta,
		Value: &value,
	}

	m.Reset()

	assert.Empty(t, m.ID)
	assert.Empty(t, m.MType)
	// Указатели не зануляются, обнуляется значение по указателю.
	require.NotNil(t, m.Delta)
	require.NotNil(t, m.Value)
	assert.Equal(t, int64(0), *m.Delta)
	assert.Equal(t, 0.0, *m.Value)
}

func TestMetrics_Reset_NilSafe(t *testing.T) {
	var m *Metrics
	assert.NotPanics(t, func() { m.Reset() })
}
