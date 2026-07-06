package adminapi

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
)

const DefaultReportsVisitsPageSize = 100

type ReportsVisitsOption func(*reportsVisitsOptions)

type reportsVisitsOptions struct {
	isStudent    *bool
	yearGroups   []int
	signStatuses []string
	page         int
	pageSize     int
}

func WithIsStudent(isStudent bool) ReportsVisitsOption {
	return func(opts *reportsVisitsOptions) {
		opts.isStudent = &isStudent
	}
}

func WithYearGroup(yearGroup int) ReportsVisitsOption {
	return func(opts *reportsVisitsOptions) {
		opts.yearGroups = []int{yearGroup}
	}
}

func WithSignStatus(signStatus string) ReportsVisitsOption {
	return func(opts *reportsVisitsOptions) {
		opts.signStatuses = []string{signStatus}
	}
}

func WithPage(page int) ReportsVisitsOption {
	return func(opts *reportsVisitsOptions) {
		opts.page = page
	}
}

func WithLimit(pageSize int) ReportsVisitsOption {
	return func(opts *reportsVisitsOptions) {
		opts.pageSize = pageSize
	}
}

var (
	ErrInvalidRequestFormat = errors.New("invalid request format")
)

type ReportsVisitsResponseItem struct {
	VisitDate       string     `json:"visit_date"`
	VisitorID       int        `json:"visitor_id"`
	Name            string     `json:"name"`
	Surname         string     `json:"surname"`
	IsStudent       bool       `json:"is_student"`
	YearGroup       *int       `json:"year_group,omitempty"`
	VisitsCount     int        `json:"visits_count"`
	SignStatus      string     `json:"sign_status"`
	SignedIn        *time.Time `json:"signed_in,omitempty"`
	SignedOut       *time.Time `json:"signed_out,omitempty"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
}

type ReportsVisitsResponse struct {
	Page       int                         `json:"page"`
	Limit      int                         `json:"limit"`
	Total      int                         `json:"total"`
	TotalPages int                         `json:"total_pages"`
	Data       []ReportsVisitsResponseItem `json:"data"`
}

func (a *AdminAPI) GetReportsVisits(from, to string, opt ...ReportsVisitsOption) (*ReportsVisitsResponse, error) {
	fromTime, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid 'from' date", ErrInvalidRequestFormat)
	}
	toTime, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid 'to' date", ErrInvalidRequestFormat)
	}

	opts := &reportsVisitsOptions{
		page:     1,
		pageSize: DefaultReportsVisitsPageSize,
	}
	for _, o := range opt {
		o(opts)
	}

	if err := a.validateReportsVisitsRequest(&fromTime, &toTime, opts); err != nil {
		return nil, err
	}

	if toTime.Before(fromTime) {
		toTime, fromTime = fromTime, toTime
	}

	toExclusive := toTime.AddDate(0, 0, 1)

	filter := provider.VisitReportFilter{
		IsStudent:    opts.isStudent,
		YearGroups:   opts.yearGroups,
		SignStatuses: opts.signStatuses,
		Page:         opts.page,
		PageSize:     opts.pageSize,
	}

	report, err := a.VisitDailyReportRepo.GetVisitReport(fromTime, toExclusive, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get visit report: %w", err)
	}

	items := make([]ReportsVisitsResponseItem, 0, len(report.Rows))
	for _, row := range report.Rows {
		items = append(items, ReportsVisitsResponseItem{
			VisitDate:       row.VisitDate.Format("2006-01-02"),
			VisitorID:       int(row.VisitorID),
			Name:            row.Name,
			Surname:         row.Surname,
			IsStudent:       row.IsStudent,
			YearGroup:       row.YearGroup,
			VisitsCount:     row.VisitsCount,
			SignStatus:      row.SignStatus,
			SignedIn:        row.SignedIn,
			SignedOut:       row.SignedOut,
			DurationMinutes: row.DurationMinutes,
		})
	}

	totalPages := 0
	if opts.pageSize > 0 {
		totalPages = int(math.Ceil(float64(report.Total) / float64(opts.pageSize)))
	}

	return &ReportsVisitsResponse{
		Page:       opts.page,
		Limit:      opts.pageSize,
		Total:      report.Total,
		TotalPages: totalPages,
		Data:       items,
	}, nil
}

var validSignStatuses = map[string]bool{
	"signed_in":  true,
	"signed_out": true,
	"not_signed": true,
}

func (a *AdminAPI) validateReportsVisitsRequest(from, to *time.Time, opts *reportsVisitsOptions) error {
	if from == nil || to == nil {
		return fmt.Errorf("%w: 'from' and 'to' dates are required", ErrInvalidRequestFormat)
	}

	for _, s := range opts.signStatuses {
		if !validSignStatuses[s] {
			return fmt.Errorf("%w: invalid 'sign_status' value, must be one of 'signed_in', 'signed_out', 'not_signed'", ErrInvalidRequestFormat)
		}
	}

	if opts.page <= 0 {
		return fmt.Errorf("%w: 'page' must be a positive integer", ErrInvalidRequestFormat)
	}

	if opts.isStudent != nil && !*opts.isStudent && len(opts.yearGroups) > 0 {
		return fmt.Errorf("%w: 'year_group' filter can only be used when 'is_student' is true", ErrInvalidRequestFormat)
	}

	return nil
}
