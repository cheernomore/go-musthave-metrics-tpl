package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRepo(t *testing.T) (*PostgresRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewPostgresRepository(db), mock
}

func TestPostgresRepository_Save(t *testing.T) {
	t.Run("counter", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		d := int64(5)
		mock.ExpectExec("INSERT INTO metrics").
			WithArgs("PollCount", "counter", d).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("gauge", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		v := 123.45
		mock.ExpectExec("INSERT INTO metrics").
			WithArgs("Alloc", "gauge", v).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresRepository_SaveBatch(t *testing.T) {
	repo, mock := newMockRepo(t)
	d := int64(3)
	v := 1.5

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO metrics") // counter
	mock.ExpectPrepare("INSERT INTO metrics") // gauge
	mock.ExpectExec("INSERT INTO metrics").
		WithArgs("PollCount", "counter", d).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO metrics").
		WithArgs("Alloc", "gauge", v).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SaveBatch([]models.Metrics{
		{ID: "PollCount", MType: models.Counter, Delta: &d},
		{ID: "Alloc", MType: models.Gauge, Value: &v},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_Find(t *testing.T) {
	repo, mock := newMockRepo(t)

	rows := sqlmock.NewRows([]string{"name", "type", "delta", "value"}).
		AddRow("Alloc", "gauge", nil, 123.45)
	mock.ExpectQuery("SELECT name, type, delta, value FROM metrics").
		WithArgs("Alloc", "gauge").
		WillReturnRows(rows)

	got, err := repo.Find("Alloc", "gauge")
	require.NoError(t, err)
	assert.Equal(t, "Alloc", got.ID)
	require.NotNil(t, got.Value)
	assert.Equal(t, 123.45, *got.Value)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_Find_NotFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery("SELECT name, type, delta, value FROM metrics").
		WithArgs("Missing", "gauge").
		WillReturnError(errors.New("sql: no rows in result set"))

	_, err := repo.Find("Missing", "gauge")
	require.Error(t, err)
}

func TestPostgresRepository_FindAll(t *testing.T) {
	repo, mock := newMockRepo(t)

	rows := sqlmock.NewRows([]string{"name", "type", "delta", "value"}).
		AddRow("Alloc", "gauge", nil, 1.5).
		AddRow("PollCount", "counter", int64(3), nil)
	mock.ExpectQuery("SELECT name, type, delta, value FROM metrics ORDER BY name").
		WillReturnRows(rows)

	all, err := repo.FindAll()
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIsRetriableDBError(t *testing.T) {
	assert.False(t, isRetriableDBError(nil))
	assert.False(t, isRetriableDBError(errors.New("обычная ошибка")))

	connErr := &pgconn.PgError{Code: pgerrcode.ConnectionException}
	assert.True(t, isRetriableDBError(connErr))

	otherPg := &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	assert.False(t, isRetriableDBError(otherPg))
}
