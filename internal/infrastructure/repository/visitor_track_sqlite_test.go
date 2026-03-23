package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

func TestStore_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	visitor := &entity.Visitor{
		Id:      123,
		Name:    "John",
		Surname: "Doe",
		Grade:   intPtr(10),
		Image:   "/test.jpg",
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  strPtr("KEY123"),
		SignedIn:  true,
		Visitor:   visitor,
	}

	expectedTime := time.Now()

	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true, nil, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(1, 123, "KEY123", true, nil, nil, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	result, err := repo.Store(visitTrack)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Id)
	assert.Equal(t, int32(123), result.VisitorId)
	assert.Equal(t, strPtr("KEY123"), result.VisitKey)
	assert.Equal(t, true, result.SignedIn)
	assert.Nil(t, result.AdminID)
	assert.Nil(t, result.Description)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_InsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  strPtr("KEY123"),
		SignedIn:  true,
	}

	expectedError := errors.New("insert failed")
	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true, nil, nil).
		WillReturnError(expectedError)

	result, err := repo.Store(visitTrack)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_LastInsertIdError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  strPtr("KEY123"),
		SignedIn:  true,
	}

	expectedError := errors.New("last insert id error")
	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true, nil, nil).
		WillReturnResult(sqlmock.NewErrorResult(expectedError))

	result, err := repo.Store(visitTrack)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_GetByIdError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	visitor := &entity.Visitor{
		Id:      123,
		Name:    "John",
		Surname: "Doe",
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 123,
		VisitKey:  strPtr("KEY123"),
		SignedIn:  true,
		Visitor:   visitor,
	}

	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true, nil, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	expectedError := errors.New("query failed")
	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnError(expectedError)

	result, err := repo.Store(visitTrack)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	expectedTime := time.Date(2023, 12, 10, 14, 30, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(5, 456, "KEY789", false, nil, nil, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(5)).
		WillReturnRows(rows)

	result, err := repo.GetById(5)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.Id)
	assert.Equal(t, int32(456), result.VisitorId)
	assert.Equal(t, strPtr("KEY789"), result.VisitKey)
	assert.Equal(t, false, result.SignedIn)
	assert.Nil(t, result.AdminID)
	assert.Nil(t, result.Description)
	assert.Equal(t, expectedTime.Format("2006-01-02 15:04:05"), result.CreatedAt.Format("2006-01-02 15:04:05"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_RFC3339(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	expectedTime := time.Date(2025, 12, 27, 11, 6, 22, 0, time.UTC)
	rfc3339Time := "2025-12-27T11:06:22Z"

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(5, 456, "KEY789", false, nil, nil, rfc3339Time)

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(5)).
		WillReturnRows(rows)

	result, err := repo.GetById(5)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.Id)
	assert.Equal(t, int32(456), result.VisitorId)
	assert.Equal(t, strPtr("KEY789"), result.VisitKey)
	assert.Equal(t, false, result.SignedIn)
	assert.Equal(t, expectedTime.Unix(), result.CreatedAt.Unix())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetById(999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	expectedError := errors.New("connection timeout")
	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnError(expectedError)

	result, err := repo.GetById(1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_InvalidTimeFormat(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(5, 456, "KEY789", false, nil, nil, "invalid-date")

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(5)).
		WillReturnRows(rows)

	result, err := repo.GetById(5)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parsing time")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountEventsByVisitorIdSince_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	visitor := &entity.Visitor{
		Id:      789,
		Name:    "Alice",
		Surname: "Wonder",
	}

	visitTrack := &entity.VisitTrack{
		VisitorId: 789,
		VisitKey:  strPtr("KEYOUT"),
		SignedIn:  false,
		Visitor:   visitor,
	}

	expectedTime := time.Now()

	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(789), "KEYOUT", false, nil, nil).
		WillReturnResult(sqlmock.NewResult(10, 1))

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(10, 789, "KEYOUT", false, nil, nil, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(10)).
		WillReturnRows(rows)

	result, err := repo.Store(visitTrack)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 10, result.Id)
	assert.Equal(t, false, result.SignedIn)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_WithAdminIDAndDescription(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	adminID := uuid.New()
	desc := "visitor forgot key"
	visitor := &entity.Visitor{
		Id:      123,
		Name:    "John",
		Surname: "Doe",
	}

	visitTrack := &entity.VisitTrack{
		VisitorId:   123,
		VisitKey:    strPtr("KEY123"),
		SignedIn:    true,
		Visitor:     visitor,
		AdminID:     &adminID,
		Description: &desc,
	}

	expectedTime := time.Now()

	mock.ExpectExec("INSERT INTO track").
		WithArgs(int32(123), "KEY123", true, adminID.String(), &desc).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(1, 123, "KEY123", true, adminID.String(), desc, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	result, err := repo.Store(visitTrack)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Id)
	assert.NotNil(t, result.AdminID)
	assert.Equal(t, adminID, *result.AdminID)
	assert.NotNil(t, result.Description)
	assert.Equal(t, desc, *result.Description)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_WithAdminIDAndDescription(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	adminID := uuid.New()
	desc := "manual entry"
	expectedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(5, 456, "KEY789", true, adminID.String(), desc, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(5)).
		WillReturnRows(rows)

	result, err := repo.GetById(5)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.Id)
	assert.NotNil(t, result.AdminID)
	assert.Equal(t, adminID, *result.AdminID)
	assert.NotNil(t, result.Description)
	assert.Equal(t, desc, *result.Description)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetById_InvalidAdminID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	expectedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "visitor_id", "key_id", "sign_in", "admin_id", "description", "created_at"}).
		AddRow(5, 456, "KEY789", true, "not-a-uuid", nil, expectedTime.Format("2006-01-02 15:04:05"))

	mock.ExpectQuery("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?").
		WithArgs(int64(5)).
		WillReturnRows(rows)

	result, err := repo.GetById(5)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid admin_id")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountEventsByVisitorIdSince_LargeCount(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

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

var reportDataColumns = []string{
	"visit_date", "visitor_id", "name", "surname", "is_student",
	"year_group", "visits_count", "sign_status",
	"signed_in", "signed_out", "duration_minutes",
}

func newReportFilter(page, pageSize int) provider.VisitReportFilter {
	return provider.VisitReportFilter{
		Page:     page,
		PageSize: pageSize,
	}
}

func TestGetVisitReport_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387).
		AddRow("2026-03-10", 31, "Anna", "Brown", true, 9, 1, "signed_in",
			"2026-03-10 09:02:33", nil, nil)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Rows, 2)

	row1 := result.Rows[0]
	assert.Equal(t, "2026-03-10", row1.VisitDate.Format("2006-01-02"))
	assert.Equal(t, int32(25), row1.VisitorID)
	assert.Equal(t, "John", row1.Name)
	assert.Equal(t, "Smith", row1.Surname)
	assert.True(t, row1.IsStudent)
	assert.NotNil(t, row1.YearGroup)
	assert.Equal(t, 10, *row1.YearGroup)
	assert.Equal(t, 2, row1.VisitsCount)
	assert.Equal(t, "signed_out", row1.SignStatus)
	assert.NotNil(t, row1.SignedIn)
	assert.NotNil(t, row1.SignedOut)
	assert.NotNil(t, row1.DurationMinutes)
	assert.Equal(t, 387, *row1.DurationMinutes)

	row2 := result.Rows[1]
	assert.Equal(t, int32(31), row2.VisitorID)
	assert.Equal(t, 1, row2.VisitsCount)
	assert.Equal(t, "signed_in", row2.SignStatus)
	assert.NotNil(t, row2.SignedIn)
	assert.Nil(t, row2.SignedOut)
	assert.Nil(t, row2.DurationMinutes)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_EmptyResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(sqlmock.NewRows(reportDataColumns))

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Rows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_FilterIsStudent(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	isStudent := true
	filter := provider.VisitReportFilter{
		IsStudent: &isStudent,
		Page:      1,
		PageSize:  20,
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to, true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, true, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.True(t, result.Rows[0].IsStudent)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_FilterYearGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	yearGroup := 10
	filter := provider.VisitReportFilter{
		YearGroup: &yearGroup,
		Page:      1,
		PageSize:  20,
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to, 10).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 10, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_FilterSignStatusNotSigned(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	signStatus := "not_signed"
	filter := provider.VisitReportFilter{
		SignStatus: &signStatus,
		Page:       1,
		PageSize:   20,
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 50, "Test", "User", false, nil, 0, "not_signed", nil, nil, nil)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "not_signed", result.Rows[0].SignStatus)
	assert.Equal(t, 0, result.Rows[0].VisitsCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_FilterSignStatusSignedIn(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	signStatus := "signed_in"
	filter := provider.VisitReportFilter{
		SignStatus: &signStatus,
		Page:       1,
		PageSize:   20,
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 31, "Anna", "Brown", true, 9, 1, "signed_in",
			"2026-03-10 09:02:33", nil, nil)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "signed_in", result.Rows[0].SignStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_FilterSignStatusSignedOut(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	signStatus := "signed_out"
	filter := provider.VisitReportFilter{
		SignStatus: &signStatus,
		Page:       1,
		PageSize:   20,
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "signed_out", result.Rows[0].SignStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_Pagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(3, 10)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 10, 20).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Equal(t, 25, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_CountQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	expectedError := errors.New("connection refused")
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnError(expectedError)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "count query failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_DataQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	expectedError := errors.New("query timeout")
	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnError(expectedError)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "data query failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	badRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", "not-an-int", "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(badRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "row scan failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_NullYearGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 50, "Test", "Staff", false, nil, 0, "not_signed", nil, nil, nil)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Len(t, result.Rows, 1)
	assert.Nil(t, result.Rows[0].YearGroup)
	assert.False(t, result.Rows[0].IsStudent)
	assert.Nil(t, result.Rows[0].SignedIn)
	assert.Nil(t, result.Rows[0].SignedOut)
	assert.Nil(t, result.Rows[0].DurationMinutes)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_RFC3339Timestamps(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10T00:00:00Z", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10T08:45:12Z", "2026-03-10T15:12:55Z", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "2026-03-10", result.Rows[0].VisitDate.Format("2006-01-02"))
	assert.NotNil(t, result.Rows[0].SignedIn)
	assert.NotNil(t, result.Rows[0].SignedOut)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_CombinedFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	isStudent := true
	yearGroup := 10
	signStatus := "signed_out"
	filter := provider.VisitReportFilter{
		IsStudent:  &isStudent,
		YearGroup:  &yearGroup,
		SignStatus: &signStatus,
		Page:       1,
		PageSize:   20,
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to, true, 10).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, true, 10, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Rows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_InvalidVisitDateFormat(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("invalid-date", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "visit_date parse failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_InvalidSignedInFormat(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"bad-timestamp", "2026-03-10 15:12:55", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "signed_in parse failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_InvalidSignedOutFormat(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &VisitorTrack{Connection: db}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	filter := newReportFilter(1, 20)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \\(SELECT").
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	dataRows := sqlmock.NewRows(reportDataColumns).
		AddRow("2026-03-10", 25, "John", "Smith", true, 10, 2, "signed_out",
			"2026-03-10 08:45:12", "bad-timestamp", 387)

	mock.ExpectQuery("SELECT").
		WithArgs(from, to, 20, 0).
		WillReturnRows(dataRows)

	result, err := repo.GetVisitReport(from, to, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "signed_out parse failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}
