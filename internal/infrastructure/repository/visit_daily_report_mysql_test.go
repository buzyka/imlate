package repository

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// q escapes a SQL fragment so it can be used with sqlmock's default regexp
// query matcher (which would otherwise interpret "?", "(", "*" as regex).
func q(fragment string) string {
	return regexp.QuoteMeta(fragment)
}

func newReportRepo(t *testing.T) (*VisitDailyReport, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &VisitDailyReport{Connection: db}, mock
}

var reportColumns = []string{
	"day", "visitor_id", "name", "surname", "email", "image",
	"is_student", "year_group", "visits_count", "sign_status",
	"login_time", "logout_time", "total_minutes_inside", "clear_minutes_inside",
}

// --- computeDayMetrics ---

func TestComputeDayMetrics_NoEvents(t *testing.T) {
	login, logout, total, clear, status := computeDayMetrics(nil)
	assert.Nil(t, login)
	assert.Nil(t, logout)
	assert.Nil(t, total)
	assert.Nil(t, clear)
	assert.Nil(t, status)
}

func TestComputeDayMetrics_SingleEvent(t *testing.T) {
	e := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	login, logout, total, clear, status := computeDayMetrics([]time.Time{e})
	assert.Equal(t, e, login)
	assert.Equal(t, e, logout)
	assert.Nil(t, total)
	assert.Equal(t, 0, clear)
	assert.Equal(t, "sign_in", status)
}

func TestComputeDayMetrics_TwoEvents(t *testing.T) {
	in := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	out := in.Add(30 * time.Minute)
	login, logout, total, clear, status := computeDayMetrics([]time.Time{in, out})
	assert.Equal(t, in, login)
	assert.Equal(t, out, logout)
	assert.Equal(t, 30, total)
	assert.Equal(t, 30, clear)
	assert.Equal(t, "sign_out", status)
}

func TestComputeDayMetrics_ThreeEvents_TrailingUnpaired(t *testing.T) {
	in := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	out := in.Add(30 * time.Minute)
	in2 := in.Add(50 * time.Minute)
	login, logout, total, clear, status := computeDayMetrics([]time.Time{in, out, in2})
	assert.Equal(t, in, login)
	assert.Equal(t, in2, logout)
	assert.Equal(t, 50, total) // span first->last
	assert.Equal(t, 30, clear) // only the (in,out) pair counts
	assert.Equal(t, "sign_in", status)
}

func TestComputeDayMetrics_FourEvents(t *testing.T) {
	in := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	out := in.Add(30 * time.Minute)
	in2 := in.Add(60 * time.Minute)
	out2 := in.Add(90 * time.Minute)
	login, logout, total, clear, status := computeDayMetrics([]time.Time{in, out, in2, out2})
	assert.Equal(t, in, login)
	assert.Equal(t, out2, logout)
	assert.Equal(t, 90, total)
	assert.Equal(t, 60, clear) // 30 + 30
	assert.Equal(t, "sign_out", status)
}

func TestComputeDayMetrics_UnorderedInput_SortedDefensively(t *testing.T) {
	in := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	out := in.Add(30 * time.Minute)
	in2 := in.Add(60 * time.Minute)
	out2 := in.Add(90 * time.Minute)
	// Deliberately shuffled order.
	login, logout, total, clear, status := computeDayMetrics([]time.Time{out2, in, out, in2})
	assert.Equal(t, in, login)
	assert.Equal(t, out2, logout)
	assert.Equal(t, 90, total)
	assert.Equal(t, 60, clear)
	assert.Equal(t, "sign_out", status)
}

// --- EnsureDayRows ---

func TestEnsureDayRows_AlreadyExists(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT 1 FROM visit_daily_report WHERE day = ?")).
		WithArgs("2026-03-10").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	err := repo.EnsureDayRows(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnsureDayRows_CreatesRows(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT 1 FROM visit_daily_report WHERE day = ?")).
		WithArgs("2026-03-10").
		WillReturnRows(sqlmock.NewRows([]string{"1"})) // no rows -> ErrNoRows
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WithArgs("2026-03-10").
		WillReturnResult(sqlmock.NewResult(0, 5))

	err := repo.EnsureDayRows(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnsureDayRows_LookupError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT 1 FROM visit_daily_report WHERE day = ?")).
		WillReturnError(errors.New("boom"))

	err := repo.EnsureDayRows(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

func TestEnsureDayRows_InsertError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT 1 FROM visit_daily_report WHERE day = ?")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WillReturnError(errors.New("insert failed"))

	err := repo.EnsureDayRows(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

// --- RecalculateVisitorDay ---

func TestRecalculateVisitorDay_Success(t *testing.T) {
	repo, mock := newReportRepo(t)
	in := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	out := in.Add(30 * time.Minute)
	mock.ExpectQuery(q("SELECT created_at FROM track WHERE visitor_id = ? AND created_at >= ? AND created_at < ? ORDER BY created_at")).
		WithArgs(int32(25), "2026-03-10 00:00:00", "2026-03-11 00:00:00").
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(in).AddRow(out))
	mock.ExpectExec(q("INSERT INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.RecalculateVisitorDay(25, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRecalculateVisitorDay_NoEvents(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT created_at FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}))
	mock.ExpectExec(q("INSERT INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.RecalculateVisitorDay(25, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRecalculateVisitorDay_QueryError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT created_at FROM track")).
		WillReturnError(errors.New("query failed"))

	err := repo.RecalculateVisitorDay(25, time.Now())
	assert.Error(t, err)
}

func TestRecalculateVisitorDay_ScanError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT created_at FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow("not-a-time"))

	err := repo.RecalculateVisitorDay(25, time.Now())
	assert.Error(t, err)
}

func TestRecalculateVisitorDay_UpsertError(t *testing.T) {
	repo, mock := newReportRepo(t)
	in := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(q("SELECT created_at FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(in))
	mock.ExpectExec(q("INSERT INTO visit_daily_report")).
		WillReturnError(errors.New("upsert failed"))

	err := repo.RecalculateVisitorDay(25, time.Now())
	assert.Error(t, err)
}

func TestRecalculateVisitorDay_RowsErr(t *testing.T) {
	repo, mock := newReportRepo(t)
	in := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(q("SELECT created_at FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(in).RowError(0, errors.New("row error")))

	err := repo.RecalculateVisitorDay(25, time.Now())
	assert.Error(t, err)
}

// --- FinalizeDay ---

func TestFinalizeDay_Success(t *testing.T) {
	repo, mock := newReportRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	in := day.Add(8 * time.Hour)

	// unconditionally ensure rows for all active visitors
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WithArgs("2026-03-10").
		WillReturnResult(sqlmock.NewResult(0, 0))
	// distinct visitors that have events
	mock.ExpectQuery(q("SELECT DISTINCT visitor_id FROM track WHERE created_at >= ? AND created_at < ?")).
		WithArgs("2026-03-10 00:00:00", "2026-03-11 00:00:00").
		WillReturnRows(sqlmock.NewRows([]string{"visitor_id"}).AddRow(int32(25)))
	// recalc for visitor 25
	mock.ExpectQuery(q("SELECT created_at FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(in))
	mock.ExpectExec(q("INSERT INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// finalize update
	mock.ExpectExec(q("UPDATE visit_daily_report SET finalized_at = NOW() WHERE day = ?")).
		WithArgs("2026-03-10").
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := repo.FinalizeDay(day)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFinalizeDay_EnsureError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WillReturnError(errors.New("ensure failed"))

	err := repo.FinalizeDay(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

func TestFinalizeDay_VisitorsQueryError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("SELECT DISTINCT visitor_id FROM track")).
		WillReturnError(errors.New("boom"))

	err := repo.FinalizeDay(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

func TestFinalizeDay_VisitorsScanError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("SELECT DISTINCT visitor_id FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"visitor_id"}).AddRow("not-an-int"))

	err := repo.FinalizeDay(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

func TestFinalizeDay_RecalcError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("SELECT DISTINCT visitor_id FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"visitor_id"}).AddRow(int32(25)))
	mock.ExpectQuery(q("SELECT created_at FROM track")).
		WillReturnError(errors.New("recalc query failed"))

	err := repo.FinalizeDay(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

func TestFinalizeDay_UpdateError(t *testing.T) {
	repo, mock := newReportRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("SELECT DISTINCT visitor_id FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"visitor_id"})) // no visitors with events
	mock.ExpectExec(q("UPDATE visit_daily_report SET finalized_at = NOW() WHERE day = ?")).
		WillReturnError(errors.New("update failed"))

	err := repo.FinalizeDay(day)
	assert.Error(t, err)
}

func TestFinalizeDay_VisitorsRowsErr(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectExec(q("INSERT IGNORE INTO visit_daily_report")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("SELECT DISTINCT visitor_id FROM track")).
		WillReturnRows(sqlmock.NewRows([]string{"visitor_id"}).AddRow(int32(25)).RowError(0, errors.New("row error")))

	err := repo.FinalizeDay(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

// --- PendingDaysBefore ---

func TestPendingDaysBefore_Success(t *testing.T) {
	repo, mock := newReportRepo(t)
	d1 := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(q("SELECT DISTINCT DATE(created_at) AS day")).
		WithArgs("2026-03-03 00:00:00", "2026-03-10 00:00:00").
		WillReturnRows(sqlmock.NewRows([]string{"day"}).AddRow(d1).AddRow(d2))

	days, err := repo.PendingDaysBefore(
		time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
	)
	assert.NoError(t, err)
	assert.Equal(t, []time.Time{d1, d2}, days)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The candidate days come from track, not from visit_daily_report, so a day
// whose report rows were never created at all is still reported as pending.
func TestPendingDaysBefore_QueriesTrackNotReportTable(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("FROM track WHERE created_at >= ? AND created_at < ?")).
		WillReturnRows(sqlmock.NewRows([]string{"day"}))

	_, err := repo.PendingDaysBefore(time.Now().AddDate(0, 0, -7), time.Now())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPendingDaysBefore_SkipsFinalizedDays(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("WHERE r.day = d.day AND r.finalized_at IS NOT NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"day"}))

	days, err := repo.PendingDaysBefore(time.Now().AddDate(0, 0, -7), time.Now())
	assert.NoError(t, err)
	assert.Empty(t, days)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPendingDaysBefore_QueryError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT DISTINCT DATE(created_at) AS day")).
		WillReturnError(errors.New("boom"))

	days, err := repo.PendingDaysBefore(time.Now().AddDate(0, 0, -7), time.Now())
	assert.Error(t, err)
	assert.Nil(t, days)
}

func TestPendingDaysBefore_ScanError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT DISTINCT DATE(created_at) AS day")).
		WillReturnRows(sqlmock.NewRows([]string{"day"}).AddRow("not-a-time"))

	days, err := repo.PendingDaysBefore(time.Now().AddDate(0, 0, -7), time.Now())
	assert.Error(t, err)
	assert.Nil(t, days)
}

func TestPendingDaysBefore_RowsErr(t *testing.T) {
	repo, mock := newReportRepo(t)
	d1 := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(q("SELECT DISTINCT DATE(created_at) AS day")).
		WillReturnRows(sqlmock.NewRows([]string{"day"}).AddRow(d1).RowError(0, errors.New("row error")))

	_, err := repo.PendingDaysBefore(time.Now().AddDate(0, 0, -7), time.Now())
	assert.Error(t, err)
}

// --- GetVisitReport ---

func expectCount(mock sqlmock.Sqlmock, total int) {
	mock.ExpectQuery(q("SELECT COUNT(*) FROM visit_daily_report")).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(total))
}

func baseFilter() provider.VisitReportFilter {
	return provider.VisitReportFilter{Page: 1, PageSize: 20}
}

func TestGetVisitReport_Success(t *testing.T) {
	repo, mock := newReportRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	in := day.Add(8 * time.Hour)
	out := day.Add(14 * time.Hour)

	expectCount(mock, 1)
	mock.ExpectQuery(q("r.clear_minutes_inside")).
		WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(
			day, int32(25), "John", "Smith", "j@x.io", "img.png",
			true, int64(10), 2, "signed_out",
			in, out, int64(360), int64(300),
		))

	res, err := repo.GetVisitReport(day, day.AddDate(0, 0, 1), baseFilter())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Total)
	require.Len(t, res.Rows, 1)
	row := res.Rows[0]
	assert.Equal(t, int32(25), row.VisitorID)
	assert.Equal(t, "signed_out", row.SignStatus)
	assert.Equal(t, "j@x.io", row.Email)
	require.NotNil(t, row.YearGroup)
	assert.Equal(t, 10, *row.YearGroup)
	require.NotNil(t, row.SignedIn)
	require.NotNil(t, row.SignedOut)
	require.NotNil(t, row.DurationMinutes)
	assert.Equal(t, 360, *row.DurationMinutes)
	require.NotNil(t, row.ClearMinutesInside)
	assert.Equal(t, 300, *row.ClearMinutesInside)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVisitReport_NullFields(t *testing.T) {
	repo, mock := newReportRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	expectCount(mock, 1)
	mock.ExpectQuery(q("r.clear_minutes_inside")).
		WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(
			day, int32(7), "No", "Show", nil, nil,
			false, nil, 0, "not_signed",
			nil, nil, nil, nil,
		))

	res, err := repo.GetVisitReport(day, day.AddDate(0, 0, 1), baseFilter())
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	row := res.Rows[0]
	assert.Equal(t, "not_signed", row.SignStatus)
	assert.Nil(t, row.YearGroup)
	assert.Nil(t, row.SignedIn)
	assert.Nil(t, row.SignedOut)
	assert.Nil(t, row.DurationMinutes)
	assert.Nil(t, row.ClearMinutesInside)
	assert.Equal(t, "", row.Email)
}

func TestGetVisitReport_EmptyResult(t *testing.T) {
	repo, mock := newReportRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	expectCount(mock, 0)
	mock.ExpectQuery(q("r.clear_minutes_inside")).
		WillReturnRows(sqlmock.NewRows(reportColumns))

	res, err := repo.GetVisitReport(day, day.AddDate(0, 0, 1), baseFilter())
	require.NoError(t, err)
	assert.Equal(t, 0, res.Total)
	assert.Empty(t, res.Rows)
}

func TestGetVisitReport_CountError(t *testing.T) {
	repo, mock := newReportRepo(t)
	mock.ExpectQuery(q("SELECT COUNT(*) FROM visit_daily_report")).
		WillReturnError(errors.New("count failed"))

	res, err := repo.GetVisitReport(time.Now(), time.Now(), baseFilter())
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestGetVisitReport_DataQueryError(t *testing.T) {
	repo, mock := newReportRepo(t)
	expectCount(mock, 3)
	mock.ExpectQuery(q("r.clear_minutes_inside")).
		WillReturnError(errors.New("data failed"))

	res, err := repo.GetVisitReport(time.Now(), time.Now(), baseFilter())
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestGetVisitReport_ScanError(t *testing.T) {
	repo, mock := newReportRepo(t)
	expectCount(mock, 1)
	mock.ExpectQuery(q("r.clear_minutes_inside")).
		WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(
			"not-a-time", int32(25), "John", "Smith", "e", "i",
			true, int64(10), 2, "signed_out",
			nil, nil, nil, nil,
		))

	res, err := repo.GetVisitReport(time.Now(), time.Now(), baseFilter())
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestGetVisitReport_RowsErr(t *testing.T) {
	repo, mock := newReportRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	expectCount(mock, 1)
	mock.ExpectQuery(q("r.clear_minutes_inside")).
		WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(
			day, int32(25), "John", "Smith", "e", "i",
			true, int64(10), 2, "signed_out", nil, nil, nil, nil,
		).RowError(0, errors.New("row error")))

	_, err := repo.GetVisitReport(day, day.AddDate(0, 0, 1), baseFilter())
	assert.Error(t, err)
}

// --- pure helper functions ---

func TestBuildVisitorFilters(t *testing.T) {
	isStudent := true
	clause, args := buildVisitorFilters(provider.VisitReportFilter{
		IsStudent:  &isStudent,
		YearGroups: []int{10, 11},
	})
	assert.Contains(t, clause, "v.is_student = ?")
	assert.Contains(t, clause, "v.year_group IN (?, ?)")
	assert.Equal(t, []interface{}{true, 10, 11}, args)

	clause, args = buildVisitorFilters(provider.VisitReportFilter{})
	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestBuildSignStatusWhere(t *testing.T) {
	assert.Equal(t, "", buildSignStatusWhere(nil))
	assert.Equal(t, "", buildSignStatusWhere([]string{"unknown"}))
	assert.Equal(t, " AND r.visits_count = 0", buildSignStatusWhere([]string{"not_signed"}))
	assert.Equal(t, " AND (r.visits_count > 0 AND r.last_status = 'sign_in')", buildSignStatusWhere([]string{"signed_in"}))
	multi := buildSignStatusWhere([]string{"signed_in", "signed_out"})
	assert.Contains(t, multi, " AND (")
	assert.Contains(t, multi, " OR ")
}

func TestBuildOrderClause(t *testing.T) {
	assert.Equal(t, " ORDER BY r.day, v.surname, r.visitor_id", buildOrderClause(provider.VisitReportFilter{}))
	assert.Equal(t, " ORDER BY r.day, v.surname, r.visitor_id", buildOrderClause(provider.VisitReportFilter{OrderField: "bogus"}))
	assert.Equal(t, " ORDER BY v.year_group ASC, r.day ASC, r.visitor_id", buildOrderClause(provider.VisitReportFilter{OrderField: "year_group", OrderDirection: "asc"}))
	assert.Equal(t, " ORDER BY sign_status DESC, r.day DESC, r.visitor_id", buildOrderClause(provider.VisitReportFilter{OrderField: "sign_status", OrderDirection: "desc"}))
	assert.Equal(t, " ORDER BY r.visits_count ASC, r.day ASC, r.visitor_id", buildOrderClause(provider.VisitReportFilter{OrderField: "visits_count"}))
}

// Sorting by visit_date must not repeat r.day; it only needs the unique tie-breaker.
func TestBuildOrderClause_VisitDateDoesNotRepeatDay(t *testing.T) {
	assert.Equal(t, " ORDER BY r.day ASC, r.visitor_id", buildOrderClause(provider.VisitReportFilter{OrderField: "visit_date"}))
	assert.Equal(t, " ORDER BY r.day DESC, r.visitor_id", buildOrderClause(provider.VisitReportFilter{OrderField: "visit_date", OrderDirection: "desc"}))
}

// Every clause must end in r.visitor_id so paging cannot repeat or skip rows.
func TestBuildOrderClause_AlwaysHasUniqueTieBreaker(t *testing.T) {
	for field := range reportSortSQL {
		for _, dir := range []string{"asc", "desc"} {
			clause := buildOrderClause(provider.VisitReportFilter{OrderField: field, OrderDirection: dir})
			assert.True(t, strings.HasSuffix(clause, ", r.visitor_id"), "field %s dir %s: %s", field, dir, clause)
		}
	}
	assert.True(t, strings.HasSuffix(buildOrderClause(provider.VisitReportFilter{}), ", r.visitor_id"))
}

// exercise the filter/sign-status/order args flowing through GetVisitReport
func TestGetVisitReport_WithFiltersAndOrder(t *testing.T) {
	repo, mock := newReportRepo(t)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	isStudent := true
	filter := provider.VisitReportFilter{
		IsStudent:      &isStudent,
		YearGroups:     []int{10},
		SignStatuses:   []string{"signed_in", "signed_out"},
		OrderField:     "surname",
		OrderDirection: "desc",
		Page:           2,
		PageSize:       5,
	}

	mock.ExpectQuery(q("SELECT COUNT(*) FROM visit_daily_report")).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(6))
	mock.ExpectQuery(q("ORDER BY v.surname DESC")).
		WillReturnRows(sqlmock.NewRows(reportColumns))

	res, err := repo.GetVisitReport(day, day.AddDate(0, 0, 1), filter)
	require.NoError(t, err)
	assert.Equal(t, 6, res.Total)
	assert.NoError(t, mock.ExpectationsWereMet())
}
