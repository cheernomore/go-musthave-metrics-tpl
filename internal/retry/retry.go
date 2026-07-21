// Package retry предоставляет выполнение операций с повторными попытками
// при временных (retriable) ошибках.
package retry

import (
	"fmt"
	"time"
)

// RetryableFunc — функция, выполнение которой можно повторить.
type RetryableFunc func() error

// ShouldRetryFunc определяет, является ли ошибка retriable (повторяемой).
type ShouldRetryFunc func(error) bool

// DefaultIntervals — интервалы между повторами по умолчанию.
var DefaultIntervals = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

// WithRetry выполняет fn, повторяя её при retriable-ошибках. Число повторов и
// паузы между ними задаются intervals; если intervals не переданы, берутся
// DefaultIntervals (1s, 3s, 5s). Передача малых интервалов (или нулевых)
// позволяет тестировать логику повторов без реальных задержек.
func WithRetry(fn RetryableFunc, shouldRetry ShouldRetryFunc, intervals ...time.Duration) error {
	if len(intervals) == 0 {
		intervals = DefaultIntervals
	}

	// Первая попытка.
	err := fn()
	if err == nil {
		return nil
	}
	if !shouldRetry(err) {
		return err
	}

	// Дополнительные попытки с интервалами.
	for _, interval := range intervals {
		time.Sleep(interval)

		err = fn()
		if err == nil {
			return nil
		}
		if !shouldRetry(err) {
			return err
		}
	}

	return fmt.Errorf("all retry attempts exhausted: %w", err)
}
