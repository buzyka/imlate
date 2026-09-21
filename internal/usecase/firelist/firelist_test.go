package firelist

import (
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/buzyka/imlate/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newService returns a service whose alarm is already running, because every
// test below is about what the roster does once it is allowed to be read. The
// gate itself is covered separately by the TestGetFireList_Alarm* tests.
func newService() (*providertest.VisitDailyReportRepositoryMock, *Service) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	alarm := &entity.FireAlarmState{}
	if err := alarm.Enable(time.Hour); err != nil {
		panic(err)
	}
	return repo, &Service{Reports: repo, Alarm: alarm, Logger: zap.NewNop().Sugar()}
}

func row(id int32, name, surname, status string) provider.VisitReportRow {
	return provider.VisitReportRow{
		VisitorID:  id,
		Name:       name,
		Surname:    surname,
		SignStatus: status,
	}
}

func result(rows ...provider.VisitReportRow) *provider.VisitReportResult {
	return &provider.VisitReportResult{Total: len(rows), Rows: rows}
}

func surnames(rows []FireListRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Surname)
	}
	return out
}

func TestGetFireList_GradeParsing(t *testing.T) {
	tests := []struct {
		name       string
		gradeParam string
		wantLabel  string
		assertFn   func(t *testing.T, f provider.VisitReportFilter)
	}{
		{
			name:       "numeric grade filters students of that year group",
			gradeParam: "5",
			wantLabel:  "Grade 5",
			assertFn: func(t *testing.T, f provider.VisitReportFilter) {
				require.NotNil(t, f.IsStudent)
				assert.True(t, *f.IsStudent)
				assert.Equal(t, []int{5}, f.YearGroups)
			},
		},
		{
			name:       "staff filters non-students with no year group",
			gradeParam: "staff",
			wantLabel:  "Staff",
			assertFn: func(t *testing.T, f provider.VisitReportFilter) {
				require.NotNil(t, f.IsStudent)
				assert.False(t, *f.IsStudent)
				assert.Empty(t, f.YearGroups)
			},
		},
		{
			// The link may be typed by hand on a phone keyboard that capitalises.
			name:       "staff is case-insensitive",
			gradeParam: "STAFF",
			wantLabel:  "Staff",
			assertFn: func(t *testing.T, f provider.VisitReportFilter) {
				require.NotNil(t, f.IsStudent)
				assert.False(t, *f.IsStudent)
				assert.Empty(t, f.YearGroups)
			},
		},
		{
			name:       "double digit grade",
			gradeParam: "12",
			wantLabel:  "Grade 12",
			assertFn: func(t *testing.T, f provider.VisitReportFilter) {
				assert.Equal(t, []int{12}, f.YearGroups)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, svc := newService()
			var got provider.VisitReportFilter

			repo.On("EnsureDayRows", mock.Anything).Return(nil)
			repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.MatchedBy(
				func(f provider.VisitReportFilter) bool {
					got = f
					return true
				})).Return(result(), nil)

			data, err := svc.GetFireList(tt.gradeParam)

			require.NoError(t, err)
			assert.Equal(t, tt.gradeParam, data.GradeParam)
			assert.Equal(t, tt.wantLabel, data.GradeLabel)
			tt.assertFn(t, got)
			repo.AssertExpectations(t)
		})
	}
}

func TestGetFireList_InvalidGrade(t *testing.T) {
	for _, gradeParam := range []string{"abc", "", "5a", "-", "1.5"} {
		t.Run("grade="+gradeParam, func(t *testing.T) {
			repo, svc := newService()

			data, err := svc.GetFireList(gradeParam)

			require.ErrorIs(t, err, ErrInvalidGrade)
			assert.Equal(t, FireListPageData{}, data)
			// A bad URL segment must never reach the database.
			repo.AssertNotCalled(t, "EnsureDayRows", mock.Anything)
			repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestGetFireList_Pagination(t *testing.T) {
	repo, svc := newService()
	var got provider.VisitReportFilter

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.MatchedBy(
		func(f provider.VisitReportFilter) bool {
			got = f
			return true
		})).Return(result(), nil)

	_, err := svc.GetFireList("5")

	require.NoError(t, err)
	assert.Equal(t, 1, got.Page)
	// The whole class must be on one page; paging during an evacuation is
	// actively harmful.
	assert.Equal(t, maxFireListRows, got.PageSize)
	assert.Empty(t, got.OrderField, "ordering is done in Go, not SQL")
}

// Regression guard: the admin SPA's helper spans two days because it adds a day
// to `to`, which is why it has to dedupe rows client-side.
func TestGetFireList_DateRangeIsExactlyOneDay(t *testing.T) {
	repo, svc := newService()
	var from, to time.Time
	var ensuredDay time.Time

	repo.On("EnsureDayRows", mock.MatchedBy(func(d time.Time) bool {
		ensuredDay = d
		return true
	})).Return(nil)
	repo.On("GetVisitReport",
		mock.MatchedBy(func(f time.Time) bool { from = f; return true }),
		mock.MatchedBy(func(t2 time.Time) bool { to = t2; return true }),
		mock.Anything,
	).Return(result(), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	assert.Equal(t, 24*time.Hour, to.Sub(from))
	assert.Equal(t, from, ensuredDay, "the ensured day must be the day being read")
	assert.Equal(t, from.Format("2006-01-02"), data.Day)
	assert.Equal(t, from.Format("15:04"), data.GeneratedAt)
}

func TestGetFireList_PrioritySort(t *testing.T) {
	repo, svc := newService()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(
		row(1, "Zoe", "Yates", "signed_out"),
		row(2, "Bob", "Adams", "not_signed"),
		row(3, "Alice", "Smith", "signed_in"),
		row(4, "Carla", "Adams", "signed_in"),
		row(5, "Amy", "Adams", "signed_in"),
		row(6, "Dan", "Brown", "signed_out"),
		row(7, "Eve", "Zulu", "signed_in"),
	), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	assert.Equal(t, []int32{5, 4, 3, 7, 6, 1, 2},
		func() []int32 {
			ids := make([]int32, 0, len(data.Rows))
			for _, r := range data.Rows {
				ids = append(ids, r.VisitorID)
			}
			return ids
		}(),
		"signed_in (Adams/Amy, Adams/Carla, Smith, Zulu), then signed_out (Brown, Yates), then not_signed")
}

// Names like Muñecas and Thézé sort after "Z" with a naive lowercase compare,
// which breaks a teacher scanning the list alphabetically.
func TestGetFireList_DiacriticSort(t *testing.T) {
	repo, svc := newService()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(
		row(1, "A", "Nauta", "signed_in"),
		row(2, "B", "Muñecas", "signed_in"),
		row(3, "C", "Muller", "signed_in"),
		row(4, "D", "Thézé", "signed_in"),
		row(5, "E", "Zimmer", "signed_in"),
	), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	assert.Equal(t, []string{"Muller", "Muñecas", "Nauta", "Thézé", "Zimmer"}, surnames(data.Rows))
}

// Case must not decide the order: a lowercase "van Dijk" belongs among the V's,
// not before every capitalised surname (which is what a byte compare would do).
func TestGetFireList_CaseInsensitiveSort(t *testing.T) {
	repo, svc := newService()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(
		row(1, "A", "Wood", "signed_in"),
		row(2, "B", "van Dijk", "signed_in"),
		row(3, "C", "Ubank", "signed_in"),
	), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	assert.Equal(t, []string{"Ubank", "van Dijk", "Wood"}, surnames(data.Rows))
}

func TestGetFireList_StatusMapping(t *testing.T) {
	repo, svc := newService()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(
		row(1, "A", "A", "signed_in"),
		row(2, "B", "B", "signed_out"),
		row(3, "C", "C", "not_signed"),
		// An unknown status must not vanish: it is shown as unaccounted for.
		row(4, "D", "D", "something_else"),
	), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	require.Len(t, data.Rows, 4)
	assert.Equal(t, FireListRow{VisitorID: 1, Name: "A", Surname: "A", Status: "signed_in", StatusLabel: "Signed In", StatusRank: 0}, data.Rows[0])
	assert.Equal(t, FireListRow{VisitorID: 2, Name: "B", Surname: "B", Status: "signed_out", StatusLabel: "Signed Out", StatusRank: 1}, data.Rows[1])
	assert.Equal(t, FireListRow{VisitorID: 3, Name: "C", Surname: "C", Status: "not_signed", StatusLabel: "No Status", StatusRank: 2}, data.Rows[2])
	assert.Equal(t, FireListRow{VisitorID: 4, Name: "D", Surname: "D", Status: "not_signed", StatusLabel: "No Status", StatusRank: 2}, data.Rows[3])
}

func TestGetFireList_Counts(t *testing.T) {
	repo, svc := newService()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(
		row(1, "A", "A", "signed_in"),
		row(2, "B", "B", "signed_in"),
		row(3, "C", "C", "signed_in"),
		row(4, "D", "D", "signed_out"),
		row(5, "E", "E", "signed_out"),
		row(6, "F", "F", "not_signed"),
	), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	assert.Equal(t, FireListCounts{SignedIn: 3, SignedOut: 2, NoStatus: 1, Total: 6}, data.Counts)
	assert.Equal(t, version.Version, data.AppVersion)
	assert.Empty(t, data.Error)
}

func TestGetFireList_EmptyResult(t *testing.T) {
	repo, svc := newService()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	assert.Empty(t, data.Rows)
	assert.Equal(t, FireListCounts{}, data.Counts)
}

func TestGetFireList_ReportErrorIsPropagated(t *testing.T) {
	repo, svc := newService()
	wantErr := errors.New("db down")

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(nil, wantErr)

	data, err := svc.GetFireList("5")

	require.ErrorIs(t, err, wantErr)
	assert.Equal(t, FireListPageData{}, data)
}

// EnsureDayRows is best-effort: failing to seed today's rows must not stop the
// read, because rows may already exist from an earlier scan.
func TestGetFireList_EnsureDayRowsErrorIsNotFatal(t *testing.T) {
	repo, svc := newService()

	repo.On("EnsureDayRows", mock.Anything).Return(errors.New("insert failed"))
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(
		row(1, "A", "A", "signed_in"),
	), nil)

	data, err := svc.GetFireList("5")

	require.NoError(t, err)
	require.Len(t, data.Rows, 1)
	repo.AssertExpectations(t)
}

// The service is filled by the DI container, but a nil logger must not turn a
// best-effort warning into a panic on the evacuation page.
func TestGetFireList_NilLoggerDoesNotPanic(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	alarm := &entity.FireAlarmState{}
	require.NoError(t, alarm.Enable(time.Hour))
	svc := &Service{Reports: repo, Alarm: alarm}

	repo.On("EnsureDayRows", mock.Anything).Return(errors.New("insert failed"))
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).Return(result(), nil)

	assert.NotPanics(t, func() {
		_, err := svc.GetFireList("staff")
		assert.NoError(t, err)
	})
}

func TestErrorPageData(t *testing.T) {
	tests := []struct {
		name       string
		gradeParam string
		wantLabel  string
	}{
		{name: "numeric grade keeps its heading", gradeParam: "7", wantLabel: "Grade 7"},
		{name: "staff keeps its heading", gradeParam: "staff", wantLabel: "Staff"},
		{name: "mixed case staff", gradeParam: "Staff", wantLabel: "Staff"},
		// An empty heading would render an empty <h1> and a title reading
		// " — Evacuation" on the 404 page, which tells the teacher nothing.
		{name: "unparseable grade falls back to a generic heading", gradeParam: "abc", wantLabel: "Fire list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := ErrorPageData(tt.gradeParam, "boom")

			assert.Equal(t, tt.gradeParam, data.GradeParam)
			assert.Equal(t, tt.wantLabel, data.GradeLabel)
			assert.Equal(t, "boom", data.Error)
			assert.Equal(t, version.Version, data.AppVersion)
			assert.NotEmpty(t, data.Day)
			assert.NotEmpty(t, data.GeneratedAt)
			assert.Empty(t, data.Rows)
		})
	}
}

// The roster names every child in a class and says who is in the building, so
// with no alarm running nothing may reach the database at all — not even the
// EnsureDayRows write, which an anonymous request would otherwise be able to
// trigger. Asserting "no repository calls" is the real leak test; asserting
// the error alone would still pass if the query ran and the result was dropped.
func TestGetFireList_AlarmOffReturnsNoDataAndTouchesNoRepository(t *testing.T) {
	for _, gradeParam := range []string{"5", "staff", "abc"} {
		t.Run("grade="+gradeParam, func(t *testing.T) {
			repo := new(providertest.VisitDailyReportRepositoryMock)
			svc := &Service{Reports: repo, Alarm: &entity.FireAlarmState{}, Logger: zap.NewNop().Sugar()}

			data, err := svc.GetFireList(gradeParam)

			require.ErrorIs(t, err, ErrAlarmInactive)
			assert.Equal(t, FireListPageData{}, data)
			repo.AssertNotCalled(t, "EnsureDayRows", mock.Anything)
			repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// An expired alarm must close the page exactly like a switched-off one.
func TestGetFireList_ExpiredAlarmReturnsNoData(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	alarm := &entity.FireAlarmState{}
	require.NoError(t, alarm.Enable(time.Millisecond))
	svc := &Service{Reports: repo, Alarm: alarm, Logger: zap.NewNop().Sugar()}

	time.Sleep(5 * time.Millisecond)

	_, err := svc.GetFireList("5")

	require.ErrorIs(t, err, ErrAlarmInactive)
	repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
}

// The gate is checked before the grade is parsed, so a bad URL must not reveal
// that it is bad while the page is closed.
func TestGetFireList_AlarmGateOutranksGradeValidation(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	svc := &Service{Reports: repo, Alarm: &entity.FireAlarmState{}, Logger: zap.NewNop().Sugar()}

	_, err := svc.GetFireList("not-a-grade")

	require.ErrorIs(t, err, ErrAlarmInactive)
	assert.NotErrorIs(t, err, ErrInvalidGrade)
}

func TestInactivePageData(t *testing.T) {
	data := InactivePageData("7")

	assert.True(t, data.AlarmInactive)
	assert.Equal(t, "7", data.GradeParam)
	assert.Equal(t, "Grade 7", data.GradeLabel)
	assert.Equal(t, version.Version, data.AppVersion)
	assert.NotEmpty(t, data.Day)
	assert.NotEmpty(t, data.GeneratedAt)
	// Nothing about anyone may ride along on the page that refuses the roster.
	assert.Empty(t, data.Rows)
	assert.Equal(t, FireListCounts{}, data.Counts)
	assert.Empty(t, data.Error)
}

// A container that failed to inject the alarm must leave the roster closed
// rather than panicking into a 500 or, worse, defaulting to open.
func TestGetFireList_NilAlarmClosesTheRoster(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	svc := &Service{Reports: repo, Logger: zap.NewNop().Sugar()}

	_, err := svc.GetFireList("5")

	require.ErrorIs(t, err, ErrAlarmInactive)
	repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
}
