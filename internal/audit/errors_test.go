package audit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileObserver_WriteError(t *testing.T) {
	// Каталога не существует — открыть файл не удастся.
	o := NewFileObserver(filepath.Join(t.TempDir(), "нет-такого-каталога", "audit.log"))
	assert.Error(t, o.Update(Event{Timestamp: 1}))
}

func TestHTTPObserver_InvalidURL(t *testing.T) {
	o := NewHTTPObserver("://некорректный-url")
	assert.Error(t, o.Update(Event{Timestamp: 1}))
}

func TestHTTPObserver_Unreachable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := ts.URL
	ts.Close() // сервер закрыт — отправка должна завершиться ошибкой

	o := NewHTTPObserver(url)
	assert.Error(t, o.Update(Event{Timestamp: 1}))
}

func TestSubject_NotifyContinuesAfterObserverError(t *testing.T) {
	failing := &stubObserver{err: errors.New("сбой приёмника")}
	ok := &stubObserver{}

	s := NewSubject()
	s.Register(failing)
	s.Register(ok)

	assert.NotPanics(t, func() { s.Notify(Event{Timestamp: 1}) })
	// Ошибка первого наблюдателя не прерывает рассылку второму.
	assert.Len(t, ok.events, 1)
}
