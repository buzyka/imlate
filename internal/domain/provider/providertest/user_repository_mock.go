package providertest

import (
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type UserRepositoryMock struct {
	mock.Mock
}

func (m *UserRepositoryMock) FindByID(id uuid.UUID) (*entity.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *UserRepositoryMock) FindByUsername(username string) (*entity.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *UserRepositoryMock) FindAll() ([]*entity.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.User), args.Error(1)
}

func (m *UserRepositoryMock) Create(user *entity.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *UserRepositoryMock) Update(user *entity.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *UserRepositoryMock) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}
