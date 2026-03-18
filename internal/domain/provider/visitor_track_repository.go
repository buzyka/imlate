package provider

import (
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
)

type VisitReportFilter struct {
	IsStudent  *bool
	YearGroup  *int
	SignStatus *string
	Page       int
	PageSize   int
}

type VisitReportRow struct {
	VisitDate       time.Time
	VisitorID       int32
	Name            string
	Surname         string
	IsStudent       bool
	YearGroup       *int
	VisitsCount     int
	SignStatus      string
	SignedIn        *time.Time
	SignedOut       *time.Time
	DurationMinutes *int
}

type VisitReportResult struct {
	Total int
	Rows  []VisitReportRow
}

type VisitorTrackRepository interface {
	Store(vt *entity.VisitTrack) (*entity.VisitTrack, error)
	GetById(id int64) (*entity.VisitTrack, error)
	CountEventsByVisitorIdSince(visitorId int32, date time.Time) (int, error)
	GetVisitReport(from, to time.Time, filter VisitReportFilter) (*VisitReportResult, error)
}
