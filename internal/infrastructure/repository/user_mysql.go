package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/google/uuid"
)

type UserMySQL struct {
	Connection *sql.DB `container:"type"`
}

func (r *UserMySQL) FindByID(id uuid.UUID) (*entity.User, error) {
	row := r.Connection.QueryRow(
		"SELECT id, username, password, name, surname, role, is_active, created_at, updated_at, deleted_at FROM users WHERE id = ? AND deleted_at IS NULL",
		id.String(),
	)
	return r.scanUser(row)
}

func (r *UserMySQL) FindByUsername(username string) (*entity.User, error) {
	row := r.Connection.QueryRow(
		"SELECT id, username, password, name, surname, role, is_active, created_at, updated_at, deleted_at FROM users WHERE username = ? AND deleted_at IS NULL",
		username,
	)
	return r.scanUser(row)
}

func (r *UserMySQL) FindAll() ([]*entity.User, error) {
	rows, err := r.Connection.Query(
		"SELECT id, username, password, name, surname, role, is_active, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	users := []*entity.User{}
	for rows.Next() {
		user, err := r.scanFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return users, nil
}

func (r *UserMySQL) Create(user *entity.User) error {
	_, err := r.Connection.Exec(
		"INSERT INTO users (id, username, password, name, surname, role, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		user.ID.String(),
		user.UserName,
		user.Password,
		user.Name,
		user.Surname,
		string(user.Role),
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *UserMySQL) Update(user *entity.User) error {
	result, err := r.Connection.Exec(
		"UPDATE users SET username = ?, password = ?, name = ?, surname = ?, role = ?, is_active = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
		user.UserName,
		user.Password,
		user.Name,
		user.Surname,
		string(user.Role),
		user.IsActive,
		user.UpdatedAt,
		user.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already deleted")
	}
	return nil
}

func (r *UserMySQL) Delete(id uuid.UUID) error {
	result, err := r.Connection.Exec(
		"UPDATE users SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL",
		time.Now(),
		id.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already deleted")
	}
	return nil
}

func (r *UserMySQL) scanUser(row *sql.Row) (*entity.User, error) {
	var idStr string
	var role string
	var deletedAt sql.NullTime
	var updatedAt sql.NullTime
	var createdAt sql.NullTime

	user := &entity.User{}
	err := row.Scan(
		&idStr,
		&user.UserName,
		&user.Password,
		&user.Name,
		&user.Surname,
		&role,
		&user.IsActive,
		&createdAt,
		&updatedAt,
		&deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.User{}, nil
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user id: %w", err)
	}
	user.ID = parsedID
	user.Role = entity.UserRole(role)
	if createdAt.Valid {
		user.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		user.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Time
	}
	return user, nil
}

func (r *UserMySQL) scanFromRows(rows *sql.Rows) (*entity.User, error) {
	var idStr string
	var role string
	var deletedAt sql.NullTime
	var updatedAt sql.NullTime
	var createdAt sql.NullTime

	user := &entity.User{}
	err := rows.Scan(
		&idStr,
		&user.UserName,
		&user.Password,
		&user.Name,
		&user.Surname,
		&role,
		&user.IsActive,
		&createdAt,
		&updatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, err
	}

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user id: %w", err)
	}
	user.ID = parsedID
	user.Role = entity.UserRole(role)
	if createdAt.Valid {
		user.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		user.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Time
	}
	return user, nil
}
