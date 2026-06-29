package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cheernomore/go-musthave-metrics-tpl/internal/audit"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingObserver запоминает полученные события аудита.
type recordingObserver struct {
	events []audit.Event
}

func (o *recordingObserver) Update(e audit.Event) error {
	o.events = append(o.events, e)
	return nil
}

func TestMetricHandler_Updates(t *testing.T) {
	h := NewMetricHandler(repository.NewMemStorage(), nil)

	v := 1.5
	d := int64(2)
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &v},
		{ID: "PollCount", MType: models.Counter, Delta: &d},
	}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Updates(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMetricHandler_Updates_EmptyBatch(t *testing.T) {
	h := NewMetricHandler(repository.NewMemStorage(), nil)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte(`[]`)))
	rec := httptest.NewRecorder()
	h.Updates(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMetricHandler_Updates_EmitsAudit(t *testing.T) {
	obs := &recordingObserver{}
	subject := audit.NewSubject()
	subject.Register(obs)

	h := NewMetricHandler(repository.NewMemStorage(), subject)

	v := 1.5
	body, _ := json.Marshal([]models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &v}})

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("X-Real-IP", "192.168.0.42")
	rec := httptest.NewRecorder()
	h.Updates(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, obs.events, 1)
	assert.Equal(t, []string{"Alloc"}, obs.events[0].Metrics)
	assert.Equal(t, "192.168.0.42", obs.events[0].IPAddress)
	assert.NotZero(t, obs.events[0].Timestamp)
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*http.Request)
		expect string
	}{
		{
			name:   "X-Real-IP",
			setup:  func(r *http.Request) { r.Header.Set("X-Real-IP", "10.0.0.1") },
			expect: "10.0.0.1",
		},
		{
			name:   "X-Forwarded-For first",
			setup:  func(r *http.Request) { r.Header.Set("X-Forwarded-For", "10.0.0.2, 10.0.0.3") },
			expect: "10.0.0.2",
		},
		{
			name:   "RemoteAddr fallback",
			setup:  func(r *http.Request) { r.RemoteAddr = "10.0.0.4:5555" },
			expect: "10.0.0.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = ""
			tt.setup(req)
			assert.Equal(t, tt.expect, clientIP(req))
		})
	}
}
