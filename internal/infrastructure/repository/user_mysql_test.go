package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var userColumns = []string{"id", "username", "password", "name", "surname", "role", "is_active", "created_by", "created_at", "updated_at", "deleted_at"}

func newUserRepo(t *testing.T) (*UserMySQL, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	repo := &UserMySQL{Connection: db}
	return repo, mock, db
}

// ==================== FindByID ====================

func TestUserMySQL_FindByID_Success(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid.String(), "admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, now, now, nil)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE id = \\? AND deleted_at IS NULL").
		WithArgs(uid.String()).
		WillReturnRows(rows)

	// Execute
	user, err := repo.FindByID(uid)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uid, user.ID)
	assert.Equal(t, "admin", user.UserName)
	assert.Equal(t, "$2a$10$hash", user.Password)
	assert.Equal(t, "Admin", user.Name)
	assert.Equal(t, "User", user.Surname)
	assert.Equal(t, entity.UserRoleAdmin, user.Role)
	assert.True(t, user.IsActive)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
	assert.Nil(t, user.DeletedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindByID_NotFound(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE id = \\? AND deleted_at IS NULL").
		WithArgs(uid.String()).
		WillReturnError(sql.ErrNoRows)

	// Execute
	user, err := repo.FindByID(uid)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uuid.UUID{}, user.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindByID_DBError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	expectedError := errors.New("database connection error")

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE id = \\? AND deleted_at IS NULL").
		WithArgs(uid.String()).
		WillReturnError(expectedError)

	// Execute
	user, err := repo.FindByID(uid)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "database connection error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindByID_InvalidUUID(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	rows := sqlmock.NewRows(userColumns).
		AddRow("not-a-valid-uuid", "admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, time.Now(), time.Now(), nil)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE id = \\? AND deleted_at IS NULL").
		WithArgs(uid.String()).
		WillReturnRows(rows)

	// Execute
	user, err := repo.FindByID(uid)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to parse user id")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== FindByUsername ====================

func TestUserMySQL_FindByUsername_Success(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid.String(), "terminal", "$2a$10$hash", "Terminal", "User", "terminal", true, nil, now, now, nil)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE username = \\? AND deleted_at IS NULL").
		WithArgs("terminal").
		WillReturnRows(rows)

	// Execute
	user, err := repo.FindByUsername("terminal")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uid, user.ID)
	assert.Equal(t, "terminal", user.UserName)
	assert.Equal(t, entity.UserRoleTerminal, user.Role)
	assert.True(t, user.IsActive)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindByUsername_NotFound(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE username = \\? AND deleted_at IS NULL").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	// Execute
	user, err := repo.FindByUsername("nonexistent")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uuid.UUID{}, user.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindByUsername_DBError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	expectedError := errors.New("connection refused")

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE username = \\? AND deleted_at IS NULL").
		WithArgs("admin").
		WillReturnError(expectedError)

	// Execute
	user, err := repo.FindByUsername("admin")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "connection refused")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== FindAll ====================

func TestUserMySQL_FindAll_Success(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid1 := uuid.New()
	uid2 := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid1.String(), "admin", "$2a$10$hash1", "Admin", "User", "admin", true, nil, now, now, nil).
		AddRow(uid2.String(), "terminal", "$2a$10$hash2", "Terminal", "User", "terminal", true, nil, now, now, nil)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnRows(rows)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, uid1, users[0].ID)
	assert.Equal(t, "admin", users[0].UserName)
	assert.Equal(t, entity.UserRoleAdmin, users[0].Role)
	assert.Equal(t, uid2, users[1].ID)
	assert.Equal(t, "terminal", users[1].UserName)
	assert.Equal(t, entity.UserRoleTerminal, users[1].Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindAll_Empty(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows(userColumns)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnRows(rows)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, users)
	assert.Len(t, users, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindAll_DBError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	expectedError := errors.New("query failed")

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnError(expectedError)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "query failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindAll_ScanError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	// Return a row with invalid UUID to trigger scan/parse error
	rows := sqlmock.NewRows(userColumns).
		AddRow("invalid-uuid", "admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, time.Now(), time.Now(), nil)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnRows(rows)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "failed to scan user row")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindAll_RowsError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid.String(), "admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, now, now, nil).
		RowError(0, errors.New("row iteration error"))

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnRows(rows)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "rows iteration error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== Create ====================

func TestUserMySQL_Create_Success(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)
	user := &entity.User{
		ID:        uid,
		UserName:  "newuser",
		Password:  "$2a$10$somehash",
		Name:      "New",
		Surname:   "User",
		Role:      entity.UserRoleAdmin,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(uid.String(), "newuser", "$2a$10$somehash", "New", "User", "admin", true, nil, now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Execute
	err := repo.Create(user)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_Create_Error(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)
	user := &entity.User{
		ID:        uid,
		UserName:  "duplicate",
		Password:  "$2a$10$somehash",
		Name:      "Dup",
		Surname:   "User",
		Role:      entity.UserRoleTerminal,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(uid.String(), "duplicate", "$2a$10$somehash", "Dup", "User", "terminal", true, nil, now, now).
		WillReturnError(errors.New("duplicate entry"))

	// Execute
	err := repo.Create(user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate entry")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== Update ====================

func TestUserMySQL_Update_Success(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)
	user := &entity.User{
		ID:        uid,
		UserName:  "updated",
		Password:  "$2a$10$newhash",
		Name:      "Updated",
		Surname:   "User",
		Role:      entity.UserRoleAdmin,
		IsActive:  true,
		UpdatedAt: now,
	}

	mock.ExpectExec("UPDATE users SET").
		WithArgs("updated", "$2a$10$newhash", "Updated", "User", "admin", true, nil, now, uid.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Execute
	err := repo.Update(user)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_Update_NotFound(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)
	user := &entity.User{
		ID:        uid,
		UserName:  "ghost",
		Password:  "$2a$10$hash",
		Name:      "Ghost",
		Surname:   "User",
		Role:      entity.UserRoleTerminal,
		IsActive:  true,
		UpdatedAt: now,
	}

	mock.ExpectExec("UPDATE users SET").
		WithArgs("ghost", "$2a$10$hash", "Ghost", "User", "terminal", true, nil, now, uid.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Execute
	err := repo.Update(user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found or already deleted")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_Update_DBError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)
	user := &entity.User{
		ID:        uid,
		UserName:  "admin",
		Password:  "$2a$10$hash",
		Name:      "Admin",
		Surname:   "User",
		Role:      entity.UserRoleAdmin,
		IsActive:  true,
		UpdatedAt: now,
	}

	mock.ExpectExec("UPDATE users SET").
		WithArgs("admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, now, uid.String()).
		WillReturnError(errors.New("connection lost"))

	// Execute
	err := repo.Update(user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_Update_RowsAffectedError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)
	user := &entity.User{
		ID:        uid,
		UserName:  "admin",
		Password:  "$2a$10$hash",
		Name:      "Admin",
		Surname:   "User",
		Role:      entity.UserRoleAdmin,
		IsActive:  true,
		UpdatedAt: now,
	}

	mock.ExpectExec("UPDATE users SET").
		WithArgs("admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, now, uid.String()).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	// Execute
	err := repo.Update(user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== Delete ====================

func TestUserMySQL_Delete_Success(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	mock.ExpectExec("UPDATE users SET deleted_at").
		WithArgs(sqlmock.AnyArg(), uid.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Execute
	err := repo.Delete(uid)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_Delete_NotFound(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	mock.ExpectExec("UPDATE users SET deleted_at").
		WithArgs(sqlmock.AnyArg(), uid.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Execute
	err := repo.Delete(uid)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found or already deleted")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_Delete_DBError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	mock.ExpectExec("UPDATE users SET deleted_at").
		WithArgs(sqlmock.AnyArg(), uid.String()).
		WillReturnError(errors.New("delete failed"))

	// Execute
	err := repo.Delete(uid)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_Delete_RowsAffectedError(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	mock.ExpectExec("UPDATE users SET deleted_at").
		WithArgs(sqlmock.AnyArg(), uid.String()).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	// Execute
	err := repo.Delete(uid)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== scanUser edge cases ====================

func TestUserMySQL_FindByID_NullTimestamps(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid.String(), "admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE id = \\? AND deleted_at IS NULL").
		WithArgs(uid.String()).
		WillReturnRows(rows)

	// Execute
	user, err := repo.FindByID(uid)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uid, user.ID)
	assert.True(t, user.CreatedAt.IsZero())
	assert.True(t, user.UpdatedAt.IsZero())
	assert.Nil(t, user.DeletedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindByID_WithDeletedAt(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid.String(), "admin", "$2a$10$hash", "Admin", "User", "admin", false, nil, now, now, now)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE id = \\? AND deleted_at IS NULL").
		WithArgs(uid.String()).
		WillReturnRows(rows)

	// Execute
	user, err := repo.FindByID(uid)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, user.DeletedAt)
	assert.Equal(t, now, *user.DeletedAt)
	assert.False(t, user.IsActive)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ==================== scanFromRows edge cases ====================

func TestUserMySQL_FindAll_NullTimestamps(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid.String(), "admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnRows(rows)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.True(t, users[0].CreatedAt.IsZero())
	assert.True(t, users[0].UpdatedAt.IsZero())
	assert.Nil(t, users[0].DeletedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindAll_WithDeletedAt(t *testing.T) {
	// Setup
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	uid := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows(userColumns).
		AddRow(uid.String(), "admin", "$2a$10$hash", "Admin", "User", "admin", true, nil, now, now, now)

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnRows(rows)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.NotNil(t, users[0].DeletedAt)
	assert.Equal(t, now, *users[0].DeletedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQL_FindAll_RowScanError(t *testing.T) {
	// Setup — wrong number of columns triggers rows.Scan error
	repo, mock, db := newUserRepo(t)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "username"}).
		AddRow("some-id", "admin")

	mock.ExpectQuery("SELECT id, username, password, name, surname, role, is_active, created_by, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC").
		WillReturnRows(rows)

	// Execute
	users, err := repo.FindAll()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "failed to scan user row")
	assert.NoError(t, mock.ExpectationsWereMet())
}
