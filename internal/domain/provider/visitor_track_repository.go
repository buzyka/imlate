package provider

import (
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
)

type VisitReportFilter struct {
	IsStudent      *bool
	YearGroups     []int
	SignStatuses   []string
	OrderField     string
	OrderDirection string
	Page           int
	PageSize       int
}

type VisitReportRow struct {
	VisitDate       time.Time
	VisitorID       int32
	Name            string
	Surname         string
	Email           string
	Image           string
	IsStudent       bool
	YearGroup       *int
	VisitsCount     int
	SignStatus      string
	SignedIn        *time.Time
	SignedOut       *time.Time
	DurationMinutes *int
	// ClearMinutesInside is the net time spent inside for the day, excluding
	// intervals when the visitor was signed out. NULL when there were no events.
	ClearMinutesInside *int
}

type VisitReportResult struct {
	Total int
	Rows  []VisitReportRow
}

type VisitorTrackRepository interface {
	Store(vt *entity.VisitTrack) (*entity.VisitTrack, error)
	GetById(id int64) (*entity.VisitTrack, error)
	CountEventsByVisitorIdSince(visitorId int32, date time.Time) (int, error)
}
