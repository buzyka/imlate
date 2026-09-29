package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/google/uuid"
)

type RefreshTokenMySQL struct {
	Connection *sql.DB `container:"type"`
}

func (r *RefreshTokenMySQL) Save(tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.Connection.Exec(
		"INSERT INTO auth_refresh_tokens (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)",
		tokenHash,
		userID.String(),
		expiresAt.UTC(),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenMySQL) FindUserID(tokenHash string, now time.Time) (uuid.UUID, error) {
	var userIDStr string
	err := r.Connection.QueryRow(
		"SELECT user_id FROM auth_refresh_tokens WHERE token_hash = ? AND expires_at > ?",
		tokenHash,
		now.UTC(),
	).Scan(&userIDStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, provider.ErrRefreshTokenNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse refresh token user id: %w", err)
	}
	return userID, nil
}

func (r *RefreshTokenMySQL) Delete(tokenHash string) error {
	if _, err := r.Connection.Exec("DELETE FROM auth_refresh_tokens WHERE token_hash = ?", tokenHash); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenMySQL) DeleteByUserID(userID uuid.UUID) error {
	if _, err := r.Connection.Exec("DELETE FROM auth_refresh_tokens WHERE user_id = ?", userID.String()); err != nil {
		return fmt.Errorf("failed to delete user refresh tokens: %w", err)
	}
	return nil
}

func (r *RefreshTokenMySQL) DeleteExpired(now time.Time) (int, error) {
	result, err := r.Connection.Exec("DELETE FROM auth_refresh_tokens WHERE expires_at <= ?", now.UTC())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return int(rowsAffected), nil
}

func (r *RefreshTokenMySQL) CountActive(now time.Time) (int, error) {
	var count int
	err := r.Connection.QueryRow(
		"SELECT COUNT(*) FROM auth_refresh_tokens WHERE expires_at > ?",
		now.UTC(),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count refresh tokens: %w", err)
	}
	return count, nil
}
