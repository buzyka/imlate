package entity

type VisitorTrackRepository interface {
	Store(vt *VisitTrack) (*VisitTrack, error)
	GetById(id int64) (*VisitTrack, error)
}