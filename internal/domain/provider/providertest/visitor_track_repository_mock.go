package providertest

import (
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/stretchr/testify/mock"
)

type VisitorTrackRepositoryMock struct {
	mock.Mock
}

func (m *VisitorTrackRepositoryMock) Store(vt *entity.VisitTrack) (*entity.VisitTrack, error) {
	args := m.Called(vt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VisitTrack), args.Error(1)
}

func (m *VisitorTrackRepositoryMock) GetById(id int64) (*entity.VisitTrack, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VisitTrack), args.Error(1)
}

func (m *VisitorTrackRepositoryMock) CountEventsByVisitorIdSince(visitorId int32, date time.Time) (int, error) {
	args := m.Called(visitorId, date)
	return args.Int(0), args.Error(1)
}
