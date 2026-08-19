package repository

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
)

const (
	dayLayout      = "2006-01-02"
	dateTimeLayout = "2006-01-02 15:04:05"
)

// VisitDailyReport is the MySQL-backed implementation of the aggregated
// per-visitor/per-day visit report table.
type VisitDailyReport struct {
	Connection *sql.DB `container:"type"`
}

// dayBounds returns the half-open [start, next-day) DATETIME bounds for the
// day of the given time. Filtering track.created_at with a range keeps the
// query sargable (uses the idx_createdAt index) instead of wrapping the column
// in DATE(), which would force a full scan on the large track table.
func dayBounds(day time.Time) (start, end string) {
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	return dayStart.Format(dateTimeLayout), dayStart.AddDate(0, 0, 1).Format(dateTimeLayout)
}

// EnsureDayRows creates empty rows for all active visitors for the given day if
// the day has no rows yet. It is used on the hot path (first event of the day)
// and short-circuits once any row exists to avoid scanning all visitors on
// every event. Use insertMissingDayRows directly when rows must be created
// unconditionally (e.g. at finalization).
func (r *VisitDailyReport) EnsureDayRows(day time.Time) error {
	dayStr := day.Format(dayLayout)

	var exists int
	err := r.Connection.QueryRow("SELECT 1 FROM visit_daily_report WHERE day = ? LIMIT 1", dayStr).Scan(&exists)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("ensure day rows lookup failed: %w", err)
	}

	return r.insertMissingDayRows(dayStr)
}

// insertMissingDayRows inserts an empty row for every active visitor that does
// not yet have one for the day. It is idempotent (INSERT IGNORE skips existing
// (day, visitor_id) pairs) and does not touch already-populated rows.
func (r *VisitDailyReport) insertMissingDayRows(dayStr string) error {
	_, err := r.Connection.Exec(
		"INSERT IGNORE INTO visit_daily_report (day, visitor_id, visits_count) SELECT ?, v.id, 0 FROM visitors v WHERE v.deleted_at IS NULL",
		dayStr,
	)
	if err != nil {
		return fmt.Errorf("ensure day rows insert failed: %w", err)
	}
	return nil
}

// RecalculateVisitorDay recomputes a single visitor/day row from that visitor's
// track events and upserts it. finalized_at is intentionally left untouched.
func (r *VisitDailyReport) RecalculateVisitorDay(visitorID int32, day time.Time) error {
	dayStr := day.Format(dayLayout)
	start, end := dayBounds(day)

	rows, err := r.Connection.Query(
		"SELECT created_at FROM track WHERE visitor_id = ? AND created_at >= ? AND created_at < ? ORDER BY created_at",
		visitorID,
		start,
		end,
	)
	if err != nil {
		return fmt.Errorf("recalculate visitor day query failed: %w", err)
	}
	var events []time.Time
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			_ = rows.Close()
			return fmt.Errorf("recalculate visitor day scan failed: %w", err)
		}
		events = append(events, t)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("recalculate visitor day iteration failed: %w", err)
	}
	_ = rows.Close()

	login, logout, total, clear, status := computeDayMetrics(events)
	visitsCount := len(events)

	_, err = r.Connection.Exec(
		`INSERT INTO visit_daily_report
			(day, visitor_id, login_time, logout_time, total_minutes_inside, clear_minutes_inside, visits_count, last_status, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			login_time = VALUES(login_time),
			logout_time = VALUES(logout_time),
			total_minutes_inside = VALUES(total_minutes_inside),
			clear_minutes_inside = VALUES(clear_minutes_inside),
			visits_count = VALUES(visits_count),
			last_status = VALUES(last_status),
			updated_at = NOW()`,
		dayStr, visitorID, login, logout, total, clear, visitsCount, status,
	)
	if err != nil {
		return fmt.Errorf("recalculate visitor day upsert failed: %w", err)
	}
	return nil
}

// computeDayMetrics derives the aggregated metrics from a slice of event
// timestamps. Callers pass events already ordered by created_at, but the
// pairing logic depends on chronological order, so we sort defensively to keep
// the result correct regardless of the caller. The returned values are ready to
// be passed as SQL arguments (nil where the column must be NULL).
func computeDayMetrics(events []time.Time) (login, logout, total, clear, status interface{}) {
	n := len(events)
	if n == 0 {
		return nil, nil, nil, nil, nil
	}

	sort.Slice(events, func(i, j int) bool { return events[i].Before(events[j]) })

	first := events[0]
	last := events[n-1]
	login = first
	logout = last

	if n%2 == 1 {
		status = "sign_in"
	} else {
		status = "sign_out"
	}

	if n > 1 {
		total = int(last.Sub(first).Minutes())
	}

	// clear_minutes_inside: sum of full sign_in -> sign_out pairs. A trailing
	// unpaired sign-in contributes 0.
	clearMinutes := 0
	for i := 0; i+1 < n; i += 2 {
		clearMinutes += int(events[i+1].Sub(events[i]).Minutes())
	}
	clear = clearMinutes

	return login, logout, total, clear, status
}

// FinalizeDay recomputes all rows for the day from track events and marks them
// finalized. It unconditionally ensures every currently-active visitor has a
// row for the day (including visitors created mid-day, after the lazy
// initialization ran), so a finalized day is guaranteed complete.
func (r *VisitDailyReport) FinalizeDay(day time.Time) error {
	dayStr := day.Format(dayLayout)

	if err := r.insertMissingDayRows(dayStr); err != nil {
		return err
	}

	start, end := dayBounds(day)
	rows, err := r.Connection.Query("SELECT DISTINCT visitor_id FROM track WHERE created_at >= ? AND created_at < ?", start, end)
	if err != nil {
		return fmt.Errorf("finalize day visitors query failed: %w", err)
	}
	var visitorIDs []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return fmt.Errorf("finalize day scan failed: %w", err)
		}
		visitorIDs = append(visitorIDs, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("finalize day iteration failed: %w", err)
	}
	_ = rows.Close()

	for _, id := range visitorIDs {
		if err := r.RecalculateVisitorDay(id, day); err != nil {
			return err
		}
	}

	if _, err := r.Connection.Exec("UPDATE visit_daily_report SET finalized_at = NOW() WHERE day = ?", dayStr); err != nil {
		return fmt.Errorf("finalize day update failed: %w", err)
	}
	return nil
}

// PendingDaysBefore returns days in [since, before) that have track events but
// no finalized report rows yet, ordered ascending.
//
// The list of candidate days comes from the raw track table, not from
// visit_daily_report: a day whose rows were never created (the app was down,
// EnsureDayRows failed, the first event of the day was dropped) has no rows to
// look at and would otherwise never be reconciled. The derived table holds at
// most one row per day in the window, so the NOT EXISTS probe runs a handful of
// times against the primary key.
func (r *VisitDailyReport) PendingDaysBefore(since, before time.Time) ([]time.Time, error) {
	start, _ := dayBounds(since)
	end, _ := dayBounds(before)

	rows, err := r.Connection.Query(
		`SELECT d.day FROM (
			SELECT DISTINCT DATE(created_at) AS day
			FROM track WHERE created_at >= ? AND created_at < ?
		) d
		WHERE NOT EXISTS (
			SELECT 1 FROM visit_daily_report r
			WHERE r.day = d.day AND r.finalized_at IS NOT NULL
		)
		ORDER BY d.day`,
		start,
		end,
	)
	if err != nil {
		return nil, fmt.Errorf("pending days query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var days []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("pending days scan failed: %w", err)
		}
		days = append(days, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pending days iteration failed: %w", err)
	}
	return days, nil
}

const reportSelectColumns = `SELECT
	r.day,
	v.id,
	v.name,
	v.surname,
	v.email,
	v.image,
	v.is_student,
	v.year_group,
	r.visits_count,
	CASE
		WHEN r.visits_count = 0 THEN 'not_signed'
		WHEN r.last_status = 'sign_in' THEN 'signed_in'
		ELSE 'signed_out'
	END AS sign_status,
	r.login_time,
	r.logout_time,
	r.total_minutes_inside,
	r.clear_minutes_inside`

// GetVisitReport reads the paginated report directly from the aggregated table.
func (r *VisitDailyReport) GetVisitReport(from, to time.Time, filter provider.VisitReportFilter) (*provider.VisitReportResult, error) {
	visitorWhere, visitorArgs := buildVisitorFilters(filter)
	signStatusWhere := buildSignStatusWhere(filter.SignStatuses)

	fromStr := from.Format(dayLayout)
	toStr := to.Format(dayLayout)

	fromClause := " FROM visit_daily_report r JOIN visitors v ON v.id = r.visitor_id" +
		" WHERE r.day >= ? AND r.day < ? AND v.deleted_at IS NULL" + visitorWhere + signStatusWhere

	countQuery := "SELECT COUNT(*)" + fromClause
	countArgs := append([]interface{}{fromStr, toStr}, visitorArgs...)

	var total int
	if err := r.Connection.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	orderClause := buildOrderClause(filter)
	dataQuery := reportSelectColumns + fromClause + orderClause + " LIMIT ? OFFSET ?"
	offset := (filter.Page - 1) * filter.PageSize
	dataArgs := append([]interface{}{fromStr, toStr}, visitorArgs...)
	dataArgs = append(dataArgs, filter.PageSize, offset)

	rows, err := r.Connection.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("data query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []provider.VisitReportRow
	for rows.Next() {
		var row provider.VisitReportRow
		var day time.Time
		var emailRaw, imageRaw sql.NullString
		var yearGroup sql.NullInt64
		var signedIn, signedOut sql.NullTime
		var duration, clear sql.NullInt64

		if err := rows.Scan(
			&day,
			&row.VisitorID,
			&row.Name,
			&row.Surname,
			&emailRaw,
			&imageRaw,
			&row.IsStudent,
			&yearGroup,
			&row.VisitsCount,
			&row.SignStatus,
			&signedIn,
			&signedOut,
			&duration,
			&clear,
		); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		row.VisitDate = day
		row.Email = emailRaw.String
		row.Image = imageRaw.String
		if yearGroup.Valid {
			yg := int(yearGroup.Int64)
			row.YearGroup = &yg
		}
		if signedIn.Valid {
			t := signedIn.Time
			row.SignedIn = &t
		}
		if signedOut.Valid {
			t := signedOut.Time
			row.SignedOut = &t
		}
		if duration.Valid {
			d := int(duration.Int64)
			row.DurationMinutes = &d
		}
		if clear.Valid {
			c := int(clear.Int64)
			row.ClearMinutesInside = &c
		}

		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return &provider.VisitReportResult{
		Total: total,
		Rows:  result,
	}, nil
}

func buildVisitorFilters(filter provider.VisitReportFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if filter.IsStudent != nil {
		clauses = append(clauses, "v.is_student = ?")
		args = append(args, *filter.IsStudent)
	}
	if len(filter.YearGroups) > 0 {
		placeholders := make([]string, len(filter.YearGroups))
		for i, yg := range filter.YearGroups {
			placeholders[i] = "?"
			args = append(args, yg)
		}
		clauses = append(clauses, "v.year_group IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

var signStatusReportConditions = map[string]string{
	"not_signed": "r.visits_count = 0",
	"signed_in":  "(r.visits_count > 0 AND r.last_status = 'sign_in')",
	"signed_out": "(r.visits_count > 0 AND r.last_status = 'sign_out')",
}

func buildSignStatusWhere(signStatuses []string) string {
	if len(signStatuses) == 0 {
		return ""
	}

	var conditions []string
	for _, status := range signStatuses {
		if cond, ok := signStatusReportConditions[status]; ok {
			conditions = append(conditions, cond)
		}
	}

	if len(conditions) == 0 {
		return ""
	}
	if len(conditions) == 1 {
		return " AND " + conditions[0]
	}
	return " AND (" + strings.Join(conditions, " OR ") + ")"
}

var reportSortSQL = map[string]string{
	"sign_status":  "sign_status",
	"year_group":   "v.year_group",
	"name":         "v.name",
	"surname":      "v.surname",
	"visit_date":   "r.day",
	"visits_count": "r.visits_count",
}

// defaultOrderClause orders by day then surname. r.visitor_id makes the order
// total, which every ORDER BY here needs: without a unique tie-breaker MySQL is
// free to return equal rows in a different order per query, so paging through
// the result set would silently repeat and skip rows.
const defaultOrderClause = " ORDER BY r.day, v.surname, r.visitor_id"

func buildOrderClause(filter provider.VisitReportFilter) string {
	if filter.OrderField == "" {
		return defaultOrderClause
	}
	sqlExpr, ok := reportSortSQL[filter.OrderField]
	if !ok {
		return defaultOrderClause
	}
	dir := "ASC"
	if strings.EqualFold(filter.OrderDirection, "desc") {
		dir = "DESC"
	}
	// Sorting by visit_date already orders by r.day, so don't repeat it.
	if sqlExpr == "r.day" {
		return " ORDER BY r.day " + dir + ", r.visitor_id"
	}
	return " ORDER BY " + sqlExpr + " " + dir + ", r.day " + dir + ", r.visitor_id"
}
