package providertest

import (
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
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

func (m *VisitorRepositoryMock) FindAll(opts ...provider.VisitorFilterOption) ([]*entity.Visitor, error) {
	args := m.Called(opts)
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

func (m *VisitorRepositoryMock) DeleteVisitor(id int32) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *VisitorRepositoryMock) RemoveKeyFromVisitor(visitorID int32, key string) error {
	args := m.Called(visitorID, key)
	return args.Error(0)
}

func (m *VisitorRepositoryMock) FindKeysByVisitorId(visitorID int32) ([]string, error) {
	args := m.Called(visitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *VisitorRepositoryMock) UpdateVisitorImage(id int32, imagePath string) error {
	args := m.Called(id, imagePath)
	return args.Error(0)
}
