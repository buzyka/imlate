package providertest

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type RefreshTokenRepositoryMock struct {
	mock.Mock
}

func (m *RefreshTokenRepositoryMock) Save(tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	args := m.Called(tokenHash, userID, expiresAt)
	return args.Error(0)
}

func (m *RefreshTokenRepositoryMock) FindUserID(tokenHash string, now time.Time) (uuid.UUID, error) {
	args := m.Called(tokenHash, now)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *RefreshTokenRepositoryMock) Delete(tokenHash string) error {
	args := m.Called(tokenHash)
	return args.Error(0)
}

func (m *RefreshTokenRepositoryMock) DeleteByUserID(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *RefreshTokenRepositoryMock) DeleteExpired(now time.Time) (int, error) {
	args := m.Called(now)
	return args.Int(0), args.Error(1)
}

func (m *RefreshTokenRepositoryMock) CountActive(now time.Time) (int, error) {
	args := m.Called(now)
	return args.Int(0), args.Error(1)
}
