package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const testRefreshTokenHash = "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"

func newRefreshTokenRepo(t *testing.T) (*RefreshTokenMySQL, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	return &RefreshTokenMySQL{Connection: db}, mock, db
}

// ==================== Save ====================

func TestRefreshTokenMySQL_Save_Success(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	userID := uuid.New()
	expiresAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

	mock.ExpectExec("INSERT INTO auth_refresh_tokens \\(token_hash, user_id, expires_at, created_at\\) VALUES \\(\\?, \\?, \\?, \\?\\)").
		WithArgs(testRefreshTokenHash, userID.String(), expiresAt.UTC(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Save(testRefreshTokenHash, userID, expiresAt)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_Save_Error(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT INTO auth_refresh_tokens").WillReturnError(errors.New("db error"))

	err := repo.Save(testRefreshTokenHash, uuid.New(), time.Now())

	assert.EqualError(t, err, "failed to save refresh token: db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== FindUserID ====================

func TestRefreshTokenMySQL_FindUserID_Success(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery("SELECT user_id FROM auth_refresh_tokens WHERE token_hash = \\? AND expires_at > \\?").
		WithArgs(testRefreshTokenHash, now.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID.String()))

	found, err := repo.FindUserID(testRefreshTokenHash, now)

	assert.NoError(t, err)
	assert.Equal(t, userID, found)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_FindUserID_NotFound(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT user_id FROM auth_refresh_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

	found, err := repo.FindUserID(testRefreshTokenHash, time.Now())

	assert.ErrorIs(t, err, provider.ErrRefreshTokenNotFound)
	assert.Equal(t, uuid.Nil, found)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_FindUserID_QueryError(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT user_id FROM auth_refresh_tokens").WillReturnError(errors.New("db error"))

	found, err := repo.FindUserID(testRefreshTokenHash, time.Now())

	assert.EqualError(t, err, "failed to find refresh token: db error")
	assert.Equal(t, uuid.Nil, found)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_FindUserID_InvalidUserID(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT user_id FROM auth_refresh_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("not-a-uuid"))

	found, err := repo.FindUserID(testRefreshTokenHash, time.Now())

	assert.ErrorContains(t, err, "failed to parse refresh token user id")
	assert.Equal(t, uuid.Nil, found)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== Delete ====================

func TestRefreshTokenMySQL_Delete_Success(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM auth_refresh_tokens WHERE token_hash = \\?").
		WithArgs(testRefreshTokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))

	assert.NoError(t, repo.Delete(testRefreshTokenHash))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_Delete_Error(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM auth_refresh_tokens WHERE token_hash = \\?").WillReturnError(errors.New("db error"))

	assert.EqualError(t, repo.Delete(testRefreshTokenHash), "failed to delete refresh token: db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== DeleteByUserID ====================

func TestRefreshTokenMySQL_DeleteByUserID_Success(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	userID := uuid.New()
	mock.ExpectExec("DELETE FROM auth_refresh_tokens WHERE user_id = \\?").
		WithArgs(userID.String()).
		WillReturnResult(sqlmock.NewResult(0, 3))

	assert.NoError(t, repo.DeleteByUserID(userID))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_DeleteByUserID_Error(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM auth_refresh_tokens WHERE user_id = \\?").WillReturnError(errors.New("db error"))

	assert.EqualError(t, repo.DeleteByUserID(uuid.New()), "failed to delete user refresh tokens: db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== DeleteExpired ====================

func TestRefreshTokenMySQL_DeleteExpired_Success(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	now := time.Now()
	mock.ExpectExec("DELETE FROM auth_refresh_tokens WHERE expires_at <= \\?").
		WithArgs(now.UTC()).
		WillReturnResult(sqlmock.NewResult(0, 4))

	deleted, err := repo.DeleteExpired(now)

	assert.NoError(t, err)
	assert.Equal(t, 4, deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_DeleteExpired_ExecError(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM auth_refresh_tokens WHERE expires_at <= \\?").WillReturnError(errors.New("db error"))

	deleted, err := repo.DeleteExpired(time.Now())

	assert.EqualError(t, err, "failed to delete expired refresh tokens: db error")
	assert.Equal(t, 0, deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_DeleteExpired_RowsAffectedError(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM auth_refresh_tokens WHERE expires_at <= \\?").
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows error")))

	deleted, err := repo.DeleteExpired(time.Now())

	assert.EqualError(t, err, "failed to get rows affected: rows error")
	assert.Equal(t, 0, deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== CountActive ====================

func TestRefreshTokenMySQL_CountActive_Success(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM auth_refresh_tokens WHERE expires_at > \\?").
		WithArgs(now.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

	count, err := repo.CountActive(now)

	assert.NoError(t, err)
	assert.Equal(t, 7, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenMySQL_CountActive_Error(t *testing.T) {
	repo, mock, db := newRefreshTokenRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM auth_refresh_tokens").WillReturnError(errors.New("db error"))

	count, err := repo.CountActive(time.Now())

	assert.EqualError(t, err, "failed to count refresh tokens: db error")
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}
