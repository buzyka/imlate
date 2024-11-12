package entity

import "time"
import "github.com/buzyka/imlate/internal/isb/stat"

type VisitorTrackRepository interface {
	Store(vt *VisitTrack) (*VisitTrack, error)
	GetById(id int64) (*VisitTrack, error)
	CountEventsByVisitorIdSince(visitorId int32, date time.Time) (int, error)
    GetDailyStat(date time.Time) (*stat.DailyGeneral, error)
}