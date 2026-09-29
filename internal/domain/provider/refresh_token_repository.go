package provider

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrRefreshTokenNotFound is returned when a refresh token does not exist or has expired.
var ErrRefreshTokenNotFound = errors.New("refresh token not found")

// RefreshTokenRepository persists refresh tokens of user sessions.
// Tokens are addressed by their hash; the raw token is never stored.
type RefreshTokenRepository interface {
	Save(tokenHash string, userID uuid.UUID, expiresAt time.Time) error
	// FindUserID returns ErrRefreshTokenNotFound when the token is missing or expired at now.
	FindUserID(tokenHash string, now time.Time) (uuid.UUID, error)
	Delete(tokenHash string) error
	// DeleteByUserID revokes all sessions of the user.
	DeleteByUserID(userID uuid.UUID) error
	DeleteExpired(now time.Time) (int, error)
	CountActive(now time.Time) (int, error)
}
