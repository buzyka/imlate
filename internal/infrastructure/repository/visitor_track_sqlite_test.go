package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/stretchr/testify/assert"
)

func TestStore_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      123,
		Name:    "John",
		Surname: "Doe",
		Grade:   10,
		Image:   "/test.jpg",
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  "KEY123",
		SignedIn:  true,
		Visitor:   visitor,
	}

	expectedTime := time.Now()

	// Expect INSERT
	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect SELECT for GetById
	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "created_at"}).
		AddRow(1, 123, "KEY123", true, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	// Execute
	result, err := repo.Store(visitTrack)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Id)
	assert.Equal(t, int32(123), result.VisitorId)
	assert.Equal(t, "KEY123", result.VisitKey)
	assert.Equal(t, true, result.SignedIn)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_InsertError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  "KEY123",
		SignedIn:  true,
	}

	expectedError := errors.New("insert failed")
	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true).
		WillReturnError(expectedError)

	// Execute
	result, err := repo.Store(visitTrack)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_LastInsertIdError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  "KEY123",
		SignedIn:  true,
	}

	expectedError := errors.New("last insert id error")
	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true).
		WillReturnResult(sqlmock.NewErrorResult(expectedError))

	// Execute
	result, err := repo.Store(visitTrack)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_GetByIdError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      123,
		Name:    "John",
		Surname: "Doe",
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  "KEY123",
		SignedIn:  true,
		Visitor:   visitor,
	}

	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	expectedError := errors.New("query failed")
	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnError(expectedError)

	// Execute
	result, err := repo.Store(visitTrack)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	expectedTime := time.Date(2023, 12, 10, 14, 30, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "created_at"}).
		AddRow(5, 456, "KEY789", false, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(5)).
		WillReturnRows(rows)

	// Execute
	result, err := repo.GetById(5)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.Id)
	assert.Equal(t, int32(456), result.VisitorId)
	assert.Equal(t, "KEY789", result.VisitKey)
	assert.Equal(t, false, result.SignedIn)
	assert.Equal(t, expectedTime.Format("2006-01-02 15:04:05"), result.CreatedAt.Format("2006-01-02 15:04:05"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_NotFound(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	// Execute
	result, err := repo.GetById(999)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_DatabaseError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	expectedError := errors.New("connection timeout")
	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnError(expectedError)

	// Execute
	result, err := repo.GetById(1)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_InvalidTimeFormat(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "created_at"}).
		AddRow(5, 456, "KEY789", false, "invalid-date")

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(5)).
		WillReturnRows(rows)

	// Execute
	result, err := repo.GetById(5)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parsing time")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountEventsByVisitorIdSince_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	startDate := time.Date(2023, 12, 10, 0, 0, 0, 0, time.Local)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM track WHERE visitor_id = \\? AND created_at > \\?").
		WithArgs(int32(123), startDate).
		WillReturnRows(rows)

	// Execute
	count, err := repo.CountEventsByVisitorIdSince(123, startDate)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountEventsByVisitorIdSince_ZeroCount(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	startDate := time.Date(2023, 12, 10, 0, 0, 0, 0, time.Local)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM track WHERE visitor_id = \\? AND created_at > \\?").
		WithArgs(int32(456), startDate).
		WillReturnRows(rows)

	// Execute
	count, err := repo.CountEventsByVisitorIdSince(456, startDate)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountEventsByVisitorIdSince_DatabaseError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	startDate := time.Date(2023, 12, 10, 0, 0, 0, 0, time.Local)

	expectedError := errors.New("query failed")
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM track WHERE visitor_id = \\? AND created_at > \\?").
		WithArgs(int32(123), startDate).
		WillReturnError(expectedError)

	// Execute
	count, err := repo.CountEventsByVisitorIdSince(123, startDate)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountEventsByVisitorIdSince_ScanError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	startDate := time.Date(2023, 12, 10, 0, 0, 0, 0, time.Local)

	// Return invalid type for count
	rows := sqlmock.NewRows([]string{"count"}).AddRow("invalid")

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM track WHERE visitor_id = \\? AND created_at > \\?").
		WithArgs(int32(123), startDate).
		WillReturnRows(rows)

	// Execute
	count, err := repo.CountEventsByVisitorIdSince(123, startDate)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_SignedInFalse(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      789,
		Name:    "Alice",
		Surname: "Wonder",
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 789,
		VisitKey:  "KEYOUT",
		SignedIn:  false,
		Visitor:   visitor,
	}

	expectedTime := time.Now()

	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(789), "KEYOUT", false).
		WillReturnResult(sqlmock.NewResult(10, 1))

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "created_at"}).
		AddRow(10, 789, "KEYOUT", false, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(10)).
		WillReturnRows(rows)

	// Execute
	result, err := repo.Store(visitTrack)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 10, result.Id)
	assert.Equal(t, false, result.SignedIn)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountEventsByVisitorIdSince_LargeCount(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &VisitorTrack{
		Connection: db,
	}

	startDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1000)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM track WHERE visitor_id = \\? AND created_at > \\?").
		WithArgs(int32(999), startDate).
		WillReturnRows(rows)

	// Execute
	count, err := repo.CountEventsByVisitorIdSince(999, startDate)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1000, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}
