package audit

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubObserver — тестовый наблюдатель, запоминающий полученные события.
type stubObserver struct {
	events []Event
	err    error
}

func (s *stubObserver) Update(event Event) error {
	s.events = append(s.events, event)
	return s.err
}

func TestSubject_NotifyAllObservers(t *testing.T) {
	first := &stubObserver{}
	second := &stubObserver{}

	subject := NewSubject()
	subject.Register(first)
	subject.Register(second)

	event := Event{Timestamp: 100, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"}
	subject.Notify(event)

	require.Len(t, first.events, 1)
	require.Len(t, second.events, 1)
	assert.Equal(t, event, first.events[0])
	assert.Equal(t, event, second.events[0])
}

func TestSubject_HasObservers(t *testing.T) {
	var nilSubject *Subject
	assert.False(t, nilSubject.HasObservers(), "nil-Subject не должен иметь наблюдателей")

	subject := NewSubject()
	assert.False(t, subject.HasObservers())

	subject.Register(&stubObserver{})
	assert.True(t, subject.HasObservers())
}

func TestSubject_NotifyNilSafe(t *testing.T) {
	var nilSubject *Subject
	assert.NotPanics(t, func() {
		nilSubject.Notify(Event{})
	})
}

func TestFileObserver_AppendsLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := NewFileObserver(path)

	require.NoError(t, observer.Update(Event{Timestamp: 1, Metrics: []string{"Alloc"}, IPAddress: "10.0.0.1"}))
	require.NoError(t, observer.Update(Event{Timestamp: 2, Metrics: []string{"Frees"}, IPAddress: "10.0.0.2"}))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Event
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &e))
		events = append(events, e)
	}
	require.NoError(t, scanner.Err())

	require.Len(t, events, 2)
	assert.Equal(t, []string{"Alloc"}, events[0].Metrics)
	assert.Equal(t, "10.0.0.2", events[1].IPAddress)
}

func TestHTTPObserver_PostsEvent(t *testing.T) {
	var received Event
	var method string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	observer := NewHTTPObserver(ts.URL)
	event := Event{Timestamp: 42, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"}

	require.NoError(t, observer.Update(event))
	assert.Equal(t, http.MethodPost, method)
	assert.Equal(t, event, received)
}

func TestHTTPObserver_ErrorOnBadStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	observer := NewHTTPObserver(ts.URL)
	assert.Error(t, observer.Update(Event{Timestamp: 1}))
}
