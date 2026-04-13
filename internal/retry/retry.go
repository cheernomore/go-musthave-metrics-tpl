package retry

import (
	"fmt"
	"time"
)

// RetryableFunc represents a function that can be retried
type RetryableFunc func() error

// ShouldRetryFunc determines if an error is retriable
type ShouldRetryFunc func(error) bool

// WithRetry executes the given function with retry logic
// It makes 3 additional attempts after the initial one with intervals: 1s, 3s, 5s
func WithRetry(fn RetryableFunc, shouldRetry ShouldRetryFunc) error {
	// Интервалы между повторами: 1s, 3s, 5s
	retryIntervals := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	// Первая попытка
	err := fn()
	if err == nil {
		return nil
	}

	// Проверяем, нужно ли повторять
	if !shouldRetry(err) {
		return err
	}

	// Дополнительные попытки с интервалами
	for attempt, interval := range retryIntervals {
		fmt.Printf("Retry attempt %d after error: %v. Waiting %v...\n", attempt+1, err, interval)
		time.Sleep(interval)

		err = fn()
		if err == nil {
			return nil
		}

		if !shouldRetry(err) {
			return err
		}
	}

	// Все попытки исчерпаны
	return fmt.Errorf("all retry attempts exhausted: %w", err)
}