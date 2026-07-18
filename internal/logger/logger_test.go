package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	require.NoError(t, Initialize("info"))
	assert.NotNil(t, Log)

	assert.Error(t, Initialize("not-a-level"))
}

func TestRequestLogger(t *testing.T) {
	require.NoError(t, Initialize("info"))

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("body"))
	})

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()

	RequestLogger(next).ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusTeapot, rec.Code)
	assert.Equal(t, "body", rec.Body.String())
}
