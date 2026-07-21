package main

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timeoutError реализует net.Error с истёкшим таймаутом.
type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestConvertToModel_RemainingKinds(t *testing.T) {
	tests := []struct {
		name      string
		metric    Metric
		wantGauge bool
	}{
		{"uint32 counter", Metric{"C32", uint32(1), "counter"}, false},
		{"uint16 gauge", Metric{"G16", uint16(2), "gauge"}, true},
		{"uint16 counter", Metric{"C16", uint16(3), "counter"}, false},
		{"uint8 gauge", Metric{"G8", uint8(4), "gauge"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, ok := convertToModel(tt.metric)
			require.True(t, ok)
			if tt.wantGauge {
				assert.NotNil(t, m.Value)
				assert.Nil(t, m.Delta)
			} else {
				assert.NotNil(t, m.Delta)
				assert.Nil(t, m.Value)
			}
		})
	}
}

func TestIsRetriableError_NetErrors(t *testing.T) {
	assert.True(t, isRetriableError(timeoutError{}), "таймаут — retriable")
	assert.True(t, isRetriableError(&net.DNSError{IsTimeout: true}), "таймаут DNS — retriable")
	assert.True(t, isRetriableError(&net.OpError{Op: "dial", Err: syscall.ECONNRESET}))
	assert.False(t, isRetriableError(errors.New("обычная ошибка")))
	assert.False(t, isRetriableError(&net.DNSError{IsNotFound: true}))
}
