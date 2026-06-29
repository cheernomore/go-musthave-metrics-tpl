package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRetry_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := WithRetry(func() error {
		calls++
		return nil
	}, func(error) bool { return true })

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestWithRetry_NonRetriableStopsImmediately(t *testing.T) {
	calls := 0
	sentinel := errors.New("fatal")
	err := WithRetry(func() error {
		calls++
		return sentinel
	}, func(error) bool { return false })

	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, calls)
}

func TestWithRetry_RetriableThenSuccess(t *testing.T) {
	calls := 0
	err := WithRetry(func() error {
		calls++
		if calls < 2 {
			return errors.New("temporary")
		}
		return nil
	}, func(error) bool { return true })

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestWithRetry_ExhaustsAllAttempts(t *testing.T) {
	calls := 0
	err := WithRetry(func() error {
		calls++
		return errors.New("always fails")
	}, func(error) bool { return true })

	require.Error(t, err)
	// 1 первичная попытка + 3 повтора
	assert.Equal(t, 4, calls)
}
