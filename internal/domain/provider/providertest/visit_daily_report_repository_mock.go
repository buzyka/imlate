package providertest

import (
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/stretchr/testify/mock"
)

type VisitDailyReportRepositoryMock struct {
	mock.Mock
}

func (m *VisitDailyReportRepositoryMock) EnsureDayRows(day time.Time) error {
	args := m.Called(day)
	return args.Error(0)
}

func (m *VisitDailyReportRepositoryMock) RecalculateVisitorDay(visitorID int32, day time.Time) error {
	args := m.Called(visitorID, day)
	return args.Error(0)
}

func (m *VisitDailyReportRepositoryMock) FinalizeDay(day time.Time) error {
	args := m.Called(day)
	return args.Error(0)
}

func (m *VisitDailyReportRepositoryMock) UnfinalizedDaysBefore(before time.Time) ([]time.Time, error) {
	args := m.Called(before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *VisitDailyReportRepositoryMock) GetVisitReport(from, to time.Time, filter provider.VisitReportFilter) (*provider.VisitReportResult, error) {
	args := m.Called(from, to, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.VisitReportResult), args.Error(1)
}
