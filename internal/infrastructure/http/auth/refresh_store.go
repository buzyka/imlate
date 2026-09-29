package httpauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/appleboy/gin-jwt/v3/core"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/google/uuid"
)

// DBRefreshTokenStore keeps gin-jwt refresh tokens in the database, so sessions
// survive restarts and are shared between app instances. Only a SHA-256 hash of
// the token is persisted.
type DBRefreshTokenStore struct {
	Repo     provider.RefreshTokenRepository
	UserRepo provider.UserRepository
}

var _ core.TokenStore = (*DBRefreshTokenStore)(nil)

func (s *DBRefreshTokenStore) Set(_ context.Context, token string, userData any, expiry time.Time) error {
	user, ok := userData.(*entity.User)
	if !ok || user == nil || user.ID == uuid.Nil {
		return fmt.Errorf("unsupported refresh token user data: %T", userData)
	}
	return s.Repo.Save(hashRefreshToken(token), user.ID, expiry)
}

// Get resolves the token to the current user record, so a session of a user who
// was deactivated or deleted after login cannot be refreshed.
func (s *DBRefreshTokenStore) Get(_ context.Context, token string) (any, error) {
	tokenHash := hashRefreshToken(token)
	userID, err := s.Repo.FindUserID(tokenHash, time.Now())
	if err != nil {
		if errors.Is(err, provider.ErrRefreshTokenNotFound) {
			return nil, core.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load refresh token user: %w", err)
	}
	if user == nil || user.ID == uuid.Nil || !user.IsActive {
		if err := s.Repo.Delete(tokenHash); err != nil {
			return nil, err
		}
		return nil, core.ErrRefreshTokenNotFound
	}
	return user, nil
}

func (s *DBRefreshTokenStore) Delete(_ context.Context, token string) error {
	return s.Repo.Delete(hashRefreshToken(token))
}

func (s *DBRefreshTokenStore) Cleanup(_ context.Context) (int, error) {
	return s.Repo.DeleteExpired(time.Now())
}

func (s *DBRefreshTokenStore) Count(_ context.Context) (int, error) {
	return s.Repo.CountActive(time.Now())
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
