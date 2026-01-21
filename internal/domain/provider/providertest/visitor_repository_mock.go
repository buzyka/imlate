package providertest

import (
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/stretchr/testify/mock"
)

type VisitorRepositoryMock struct {
	mock.Mock
}

func (m *VisitorRepositoryMock) GetAll() ([]*entity.Visitor, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Visitor), args.Error(1)
}

func (m *VisitorRepositoryMock) FindById(id int32) (*entity.Visitor, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Visitor), args.Error(1)
}

func (m *VisitorRepositoryMock) FindByKey(key string) (*entity.VisitDetails, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VisitDetails), args.Error(1)
}

func (m *VisitorRepositoryMock) AddKeyToVisitor(visitor *entity.Visitor, key string) error {
	args := m.Called(visitor, key)
	return args.Error(0)
}

func (m *VisitorRepositoryMock) AddVisitor(visitor *entity.Visitor) error {
	args := m.Called(visitor)
	return args.Error(0)
}

func (m *VisitorRepositoryMock) SaveVisitor(visitor *entity.Visitor) error {
	args := m.Called(visitor)
	return args.Error(0)
}
