package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	s := NewServer()
	require.NotNil(t, s)
	assert.Nil(t, s.db)
}

func TestServer_Ping(t *testing.T) {
	t.Run("соединение с БД не настроено", func(t *testing.T) {
		s := NewServer()

		rec := httptest.NewRecorder()
		s.ping(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("БД доступна", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()
		mock.ExpectPing()

		s := &Server{db: db}
		rec := httptest.NewRecorder()
		s.ping(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("БД недоступна", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()
		mock.ExpectPing().WillReturnError(errors.New("БД недоступна"))

		s := &Server{db: db}
		rec := httptest.NewRecorder()
		s.ping(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
