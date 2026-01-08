package provider

import (
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
)

type VisitorTrackRepository interface {
	Store(vt *entity.VisitTrack) (*entity.VisitTrack, error)
	GetById(id int64) (*entity.VisitTrack, error)
	CountEventsByVisitorIdSince(visitorId int32, date time.Time) (int, error)
}
