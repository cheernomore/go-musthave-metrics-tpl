// Package audit реализует аудит обработанных сервером запросов с метриками
// с помощью паттерна «Наблюдатель»: Subject рассылает события аудита
// всем подписанным приёмникам (Observer).
package audit

// Event описывает событие аудита успешно обработанного запроса с метриками.
type Event struct {
	// Timestamp — unix-время события.
	Timestamp int64 `json:"ts"`
	// Metrics — наименования полученных метрик.
	Metrics []string `json:"metrics"`
	// IPAddress — IP-адрес входящего запроса.
	IPAddress string `json:"ip_address"`
}
