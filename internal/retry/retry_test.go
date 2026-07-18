package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fastIntervals — нулевые интервалы, чтобы тесты не ждали реальных задержек.
var fastIntervals = []time.Duration{0, 0, 0}

func TestWithRetry_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := WithRetry(func() error {
		calls++
		return nil
	}, func(error) bool { return true }, fastIntervals...)

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestWithRetry_NonRetriableStopsImmediately(t *testing.T) {
	calls := 0
	sentinel := errors.New("fatal")
	err := WithRetry(func() error {
		calls++
		return sentinel
	}, func(error) bool { return false }, fastIntervals...)

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
	}, func(error) bool { return true }, fastIntervals...)

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestWithRetry_ExhaustsAllAttempts(t *testing.T) {
	calls := 0
	err := WithRetry(func() error {
		calls++
		return errors.New("always fails")
	}, func(error) bool { return true }, fastIntervals...)

	require.Error(t, err)
	// 1 первичная попытка + 3 повтора (по числу интервалов).
	assert.Equal(t, 4, calls)
}

func TestWithRetry_CustomIntervalCount(t *testing.T) {
	calls := 0
	err := WithRetry(func() error {
		calls++
		return errors.New("always fails")
	}, func(error) bool { return true }, 0) // один интервал → один повтор

	require.Error(t, err)
	assert.Equal(t, 2, calls)
}

func TestWithRetry_DefaultIntervalsWhenNoneGiven(t *testing.T) {
	// Без переданных интервалов используется DefaultIntervals; ошибка
	// non-retriable завершает без задержек.
	calls := 0
	err := WithRetry(func() error {
		calls++
		return errors.New("fatal")
	}, func(error) bool { return false })

	require.Error(t, err)
	assert.Equal(t, 1, calls)
}
