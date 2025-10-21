package registrationtest

import (
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/stretchr/testify/mock"
)

type RegistratorMock struct {
	mock.Mock
}

func (m *RegistratorMock) Register(visitor *entity.Visitor, period *isams.RegistrationPeriod, status *isams.RegistrationStatus) error {
	args := m.Called(visitor, period, status)
	return args.Error(0)
}

func (m *RegistratorMock) GetRegistrationPeriodByName(name string) (*isams.RegistrationPeriod, error) {
	args := m.Called(name)
	return args.Get(0).(*isams.RegistrationPeriod), args.Error(1)
}

func (m *RegistratorMock) GetRegistrationPeriods() ([]*isams.RegistrationPeriod, error) {
	args := m.Called()
	return args.Get(0).([]*isams.RegistrationPeriod), args.Error(1)
}

func (m *RegistratorMock) GetRegistrationStatusForVisitor(period *isams.RegistrationPeriod, visitor *entity.Visitor) (*isams.RegistrationStatus, error) {
	args := m.Called(period, visitor)
	return args.Get(0).(*isams.RegistrationStatus), args.Error(1)
}

func (m *RegistratorMock) GetRegistrationStatusesForVisitor(visitor *entity.Visitor) ([]*isams.RegistrationStatus, error) {
	args := m.Called(visitor)
	return args.Get(0).([]*isams.RegistrationStatus), args.Error(1)
}
