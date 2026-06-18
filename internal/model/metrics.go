// Package models содержит доменную модель метрик, общую для сервера и агента.
package models

// Типы метрик, поддерживаемые сервисом.
const (
	// Counter — счётчик: значения накапливаются (складываются).
	Counter = "counter"
	// Gauge — измеримая величина: новое значение замещает предыдущее.
	Gauge = "gauge"
)

// Metrics описывает метрику в плоской модели. Для gauge значение передаётся
// в поле Value, для counter — приращение в поле Delta.
//
// Delta и Value объявлены через указатели, чтобы отличать заданное нулевое
// значение от незаданного: пустые поля не сериализуются (omitempty).
type Metrics struct {
	// ID — имя метрики (например, "Alloc", "PollCount").
	ID string `json:"id"`
	// MType — тип метрики: Counter или Gauge.
	MType string `json:"type"`
	// Delta — значение метрики типа counter.
	Delta *int64 `json:"delta,omitempty"`
	// Value — значение метрики типа gauge.
	Value *float64 `json:"value,omitempty"`
}
