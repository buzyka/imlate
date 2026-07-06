package provider

import "time"

// VisitDailyReportRepository manages the pre-aggregated per-visitor/per-day
// visit report table. Rows are created lazily on the day's first tracking
// event, kept up to date by a background worker, and finalized once per day by
// a cron job. GetVisitReport reads the report directly from this table.
type VisitDailyReportRepository interface {
	// EnsureDayRows creates empty rows for all active visitors for the given
	// day if the day has no rows yet (lazy initialization on first event).
	EnsureDayRows(day time.Time) error
	// RecalculateVisitorDay recomputes a single visitor/day row from track
	// events and upserts it.
	RecalculateVisitorDay(visitorID int32, day time.Time) error
	// FinalizeDay recomputes all rows for the day and marks them finalized.
	FinalizeDay(day time.Time) error
	// UnfinalizedDaysBefore returns past days (strictly before the given day)
	// that still have at least one unfinalized row, ordered ascending.
	UnfinalizedDaysBefore(before time.Time) ([]time.Time, error)
	// GetVisitReport reads the paginated report from the aggregated table.
	GetVisitReport(from, to time.Time, filter VisitReportFilter) (*VisitReportResult, error)
}
