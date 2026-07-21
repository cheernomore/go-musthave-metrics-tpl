package audit

import (
	"sync"

	"github.com/cheernomore/go-musthave-metrics-tpl/internal/logger"
	"go.uber.org/zap"
)

// Observer — наблюдатель (приёмник) аудита, который получает события.
type Observer interface {
	// Update обрабатывает очередное событие аудита.
	Update(event Event) error
}

// Subject — наблюдаемый объект. Хранит список наблюдателей и рассылает
// им события аудита (роль Subject в паттерне «Наблюдатель»).
type Subject struct {
	mu        sync.RWMutex
	observers []Observer
}

// NewSubject создаёт пустой Subject без подписчиков.
func NewSubject() *Subject {
	return &Subject{}
}

// Register подписывает наблюдателя на события аудита.
func (s *Subject) Register(o Observer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, o)
}

// HasObservers сообщает, подписан ли хотя бы один наблюдатель.
// Безопасен для вызова на nil-приёмнике (аудит отключён).
func (s *Subject) HasObservers() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.observers) > 0
}

// Notify рассылает событие всем подписанным наблюдателям. Ошибка отдельного
// наблюдателя логируется и не прерывает рассылку остальным.
func (s *Subject) Notify(event Event) {
	if s == nil {
		return
	}

	s.mu.RLock()
	observers := make([]Observer, len(s.observers))
	copy(observers, s.observers)
	s.mu.RUnlock()

	for _, o := range observers {
		if err := o.Update(event); err != nil {
			logger.Log.Error("ошибка отправки события аудита", zap.Error(err))
		}
	}
}
