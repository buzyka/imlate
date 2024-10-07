package entity

import "time"

type VisitorTrackRepository interface {
	Store(vt *VisitTrack) (*VisitTrack, error)
	GetById(id int64) (*VisitTrack, error)
	CountEventsByVisitorIdSince(visitorId int32, date time.Time) (int, error)
}