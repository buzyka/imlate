package adminapi

import (
	"fmt"
	"math"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
)

// computedFields are always included in the response regardless of the fields parameter.
var computedFields = map[string]bool{
	"visit_date":           true,
	"sign_status":          true,
	"visits_count":         true,
	"signed_in":            true,
	"signed_out":           true,
	"duration_minutes":     true,
	"clear_minutes_inside": true,
}

// AllowedVisitReportFields is the set of all fields valid in the fields parameter.
// Computed fields (in computedFields) are always returned; passing them here is allowed but redundant.
var AllowedVisitReportFields = map[string]bool{
	// optional — controlled by the fields parameter
	"visitor_id": true,
	"name":       true,
	"surname":    true,
	"is_student": true,
	"year_group": true,
	"email":      true,
	"image":      true,
	// computed — always returned; listed here so they don't trigger unknown-field errors
	"visit_date":           true,
	"sign_status":          true,
	"visits_count":         true,
	"signed_in":            true,
	"signed_out":           true,
	"duration_minutes":     true,
	"clear_minutes_inside": true,
}

var allowedSortFields = map[string]bool{
	"sign_status":  true,
	"year_group":   true,
	"name":         true,
	"surname":      true,
	"visit_date":   true,
	"visits_count": true,
}

var allowedSortDirections = map[string]bool{
	"asc":  true,
	"desc": true,
}

type PostReportsVisitsRequest struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	// Fields selects which optional fields to include in each row.
	// Computed fields (visit_date, sign_status, visits_count, signed_in, signed_out, duration_minutes)
	// are always present regardless of this list.
	// Allowed values: visitor_id, visit_date, name, surname, is_student, year_group,
	// visits_count, sign_status, signed_in, signed_out, duration_minutes, email, image.
	// Omit or leave empty to return all optional fields.
	Fields  []string                  `json:"fields"`
	Filters *PostReportsVisitsFilters `json:"filters"`
	Order   *PostReportsVisitsOrder   `json:"order"`
}

type PostReportsVisitsFilters struct {
	IsStudent *bool `json:"is_student"`
	// SignStatus filters rows by sign status. Allowed values: signed_in, signed_out, not_signed.
	SignStatus []string `json:"sign_status"`
	YearGroup  []int    `json:"year_group"`
}

type PostReportsVisitsOrder struct {
	Field     string `json:"field" enums:"sign_status,year_group,name,surname,visit_date,visits_count"`
	Direction string `json:"direction" enums:"asc,desc"`
}

type PostReportsVisitsResponse struct {
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	Total      int                      `json:"total"`
	TotalPages int                      `json:"total_pages"`
	Data       []map[string]interface{} `json:"data"`
}

func (a *AdminAPI) GetPostReportsVisits(req *PostReportsVisitsRequest) (*PostReportsVisitsResponse, error) {
	fromTime, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid 'from' date", ErrInvalidRequestFormat)
	}
	toTime, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid 'to' date", ErrInvalidRequestFormat)
	}

	optionalFields, err := resolveFields(req.Fields)
	if err != nil {
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.Limit
	if pageSize <= 0 {
		pageSize = DefaultReportsVisitsPageSize
	}

	filter := provider.VisitReportFilter{
		Page:     page,
		PageSize: pageSize,
	}

	if req.Filters != nil {
		filter.IsStudent = req.Filters.IsStudent

		for _, s := range req.Filters.SignStatus {
			if !validSignStatuses[s] {
				return nil, fmt.Errorf("%w: invalid 'sign_status' value '%s', must be one of 'signed_in', 'signed_out', 'not_signed'", ErrInvalidRequestFormat, s)
			}
		}
		filter.SignStatuses = req.Filters.SignStatus
		filter.YearGroups = req.Filters.YearGroup
	}

	if req.Order != nil {
		if !allowedSortFields[req.Order.Field] {
			return nil, fmt.Errorf("%w: unsupported sort field '%s'", ErrInvalidRequestFormat, req.Order.Field)
		}
		if !allowedSortDirections[req.Order.Direction] {
			return nil, fmt.Errorf("%w: unsupported sort direction '%s', must be 'asc' or 'desc'", ErrInvalidRequestFormat, req.Order.Direction)
		}
		filter.OrderField = req.Order.Field
		filter.OrderDirection = req.Order.Direction
	}

	if toTime.Before(fromTime) {
		toTime, fromTime = fromTime, toTime
	}
	toExclusive := toTime.AddDate(0, 0, 1)

	report, err := a.VisitDailyReportRepo.GetVisitReport(fromTime, toExclusive, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get visit report: %w", err)
	}

	data := make([]map[string]interface{}, 0, len(report.Rows))
	for _, row := range report.Rows {
		data = append(data, buildPostRowMap(row, optionalFields))
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int(math.Ceil(float64(report.Total) / float64(pageSize)))
	}

	return &PostReportsVisitsResponse{
		Page:       page,
		Limit:      pageSize,
		Total:      report.Total,
		TotalPages: totalPages,
		Data:       data,
	}, nil
}

// resolveFields returns the optional fields to include. When requested is empty, all optional
// fields are returned. Computed fields are always present and are filtered out of the result
// (they don't need to be requested explicitly).
func resolveFields(requested []string) ([]string, error) {
	if len(requested) == 0 {
		all := make([]string, 0, len(AllowedVisitReportFields)-len(computedFields))
		for f := range AllowedVisitReportFields {
			if !computedFields[f] {
				all = append(all, f)
			}
		}
		return all, nil
	}
	for _, f := range requested {
		if !AllowedVisitReportFields[f] {
			return nil, fmt.Errorf("%w: unknown field '%s'", ErrInvalidRequestFormat, f)
		}
	}
	return requested, nil
}

// buildPostRowMap builds a response row. Computed fields are always included; optional fields
// are added only when present in the fields slice and not already covered by computedFields.
func buildPostRowMap(row provider.VisitReportRow, fields []string) map[string]interface{} {
	result := map[string]interface{}{
		"visit_date":           row.VisitDate.Format("2006-01-02"),
		"sign_status":          row.SignStatus,
		"visits_count":         row.VisitsCount,
		"signed_in":            row.SignedIn,
		"signed_out":           row.SignedOut,
		"duration_minutes":     row.DurationMinutes,
		"clear_minutes_inside": row.ClearMinutesInside,
	}
	optional := map[string]interface{}{
		"visitor_id": int(row.VisitorID),
		"name":       row.Name,
		"surname":    row.Surname,
		"is_student": row.IsStudent,
		"year_group": row.YearGroup,
		"email":      row.Email,
		"image":      row.Image,
	}
	for _, f := range fields {
		if !computedFields[f] {
			result[f] = optional[f]
		}
	}
	return result
}
