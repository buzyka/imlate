package firelist

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	firelistview "github.com/buzyka/imlate/internal/usecase/firelist"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// newTestController wires a real usecase Service onto a repository mock: the
// controller field is a concrete *Service, and the golobby container fills it
// by concrete type, so introducing an interface here would diverge from what
// actually runs in production. The alarm is switched on, because every test
// using this helper is about what the handler does once the roster is allowed
// to be read; the closed case has its own test.
func newTestController() (*providertest.VisitDailyReportRepositoryMock, *FireListController) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	alarm := &entity.FireAlarmState{}
	if err := alarm.Enable(time.Hour); err != nil {
		panic(err)
	}
	return repo, &FireListController{
		FireList: &firelistview.Service{Reports: repo, Alarm: alarm, Logger: zap.NewNop().Sugar()},
		Logger:   zap.NewNop().Sugar(),
	}
}

// performRequest registers a stub template so the handler can be exercised
// without depending on the real website/firelist.html.
func performRequest(handler gin.HandlerFunc, path string) *httptest.ResponseRecorder {
	router := gin.New()
	router.SetHTMLTemplate(template.Must(template.New("firelist.html").Parse(
		`grade={{ .GradeParam }} fg={{ .FormGroupParam }} label={{ .GradeLabel }} day={{ .Day }} at={{ .GeneratedAt }} ` +
			`v={{ .AppVersion }} in={{ .Counts.SignedIn }} out={{ .Counts.SignedOut }} ` +
			`none={{ .Counts.NoStatus }} total={{ .Counts.Total }} error={{ .Error }} ` +
			`inactive={{ .AlarmInactive }}` +
			`{{ range .Rows }}|{{ .VisitorID }};{{ .Name }};{{ .Surname }};{{ .Status }};{{ .StatusLabel }};{{ .StatusRank }}{{ end }}`,
	)))
	router.GET("/firelist/:grade", handler)
	router.GET("/firelist/:grade/:formGroup", handler)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestFireListPageHandler_Success(t *testing.T) {
	repo, controller := newTestController()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.MatchedBy(
		func(f provider.VisitReportFilter) bool {
			return f.IsStudent != nil && *f.IsStudent && len(f.YearGroups) == 1 && f.YearGroups[0] == 5
		})).Return(&provider.VisitReportResult{
		Total: 2,
		Rows: []provider.VisitReportRow{
			{VisitorID: 7, Name: "Zoe", Surname: "Yates", SignStatus: "not_signed"},
			{VisitorID: 3, Name: "Alice", Surname: "Adams", SignStatus: "signed_in"},
		},
	}, nil)

	w := performRequest(controller.FireListPageHandler(), "/firelist/5")

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "grade=5")
	assert.Contains(t, body, "label=Grade 5")
	assert.Contains(t, body, "in=1 out=0 none=1 total=2")
	assert.Contains(t, body, "error=")
	// Sorted output reaches the template: signed_in first.
	assert.Contains(t, body, "|3;Alice;Adams;signed_in;Signed In;0|7;Zoe;Yates;not_signed;No Status;2")
	repo.AssertExpectations(t)
}

func TestFireListPageHandler_Staff(t *testing.T) {
	repo, controller := newTestController()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.MatchedBy(
		func(f provider.VisitReportFilter) bool {
			return f.IsStudent != nil && !*f.IsStudent && len(f.YearGroups) == 0
		})).Return(&provider.VisitReportResult{}, nil)

	w := performRequest(controller.FireListPageHandler(), "/firelist/staff")

	require.Equal(t, http.StatusOK, w.Code)
	// The non-student bucket is addressed as "staff" in the URL and headed "Staff".
	assert.Contains(t, w.Body.String(), "label=Staff")
	repo.AssertExpectations(t)
}

func TestFireListPageHandler_InvalidGrade(t *testing.T) {
	repo, controller := newTestController()

	w := performRequest(controller.FireListPageHandler(), "/firelist/abc")

	require.Equal(t, http.StatusNotFound, w.Code)
	body := w.Body.String()
	// A readable page, not a JSON error.
	assert.Contains(t, body, "error="+invalidGradeMessage)
	assert.Contains(t, body, "grade=abc")
	assert.Contains(t, body, "total=0")
	repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
}

func TestFireListPageHandler_ServiceError(t *testing.T) {
	repo, controller := newTestController()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("dial tcp 10.0.0.5:3306: connection refused"))

	w := performRequest(controller.FireListPageHandler(), "/firelist/5")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "error="+unavailableMessage)
	// The underlying error may name hosts, ports or tables; this page is public.
	assert.NotContains(t, body, "connection refused")
	assert.NotContains(t, body, "10.0.0.5")
	repo.AssertExpectations(t)
}

// Stale roster data during an evacuation is worse than a slow reload.
func TestFireListPageHandler_IsNotCacheable(t *testing.T) {
	repo, controller := newTestController()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{}, nil)

	w := performRequest(controller.FireListPageHandler(), "/firelist/5")

	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

// The header must be set even on the error paths, which return before the
// happy-path render.
func TestFireListPageHandler_IsNotCacheableOnError(t *testing.T) {
	_, controller := newTestController()

	w := performRequest(controller.FireListPageHandler(), "/firelist/abc")

	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

// The controller is filled by the DI container, but a nil logger must not turn
// a database outage into a panic.
func TestFireListPageHandler_NilLoggerDoesNotPanic(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	alarm := &entity.FireAlarmState{}
	require.NoError(t, alarm.Enable(time.Hour))
	controller := &FireListController{
		FireList: &firelistview.Service{Reports: repo, Alarm: alarm},
	}

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("boom"))

	assert.NotPanics(t, func() {
		w := performRequest(controller.FireListPageHandler(), "/firelist/5")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// The whole point of the feature: with no alarm running the public page must
// refuse, and must refuse without any roster riding along in the markup.
func TestFireListPageHandler_AlarmOffIsForbiddenAndCarriesNoRoster(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	controller := &FireListController{
		FireList: &firelistview.Service{
			Reports: repo,
			Alarm:   &entity.FireAlarmState{},
			Logger:  zap.NewNop().Sugar(),
		},
		Logger: zap.NewNop().Sugar(),
	}

	for _, path := range []string{"/firelist/5", "/firelist/staff", "/firelist/abc"} {
		t.Run(path, func(t *testing.T) {
			w := performRequest(controller.FireListPageHandler(), path)

			require.Equal(t, http.StatusForbidden, w.Code)
			body := w.Body.String()
			assert.Contains(t, body, "inactive=true")
			assert.Contains(t, body, "total=0")
			// No row markup at all: the stub renders one "|id;name;..." group per
			// row, so a single pipe would mean a person leaked through the gate.
			assert.NotContains(t, body, "|")
			// The refusal is not dressed up as a failure.
			assert.Contains(t, body, "error=")
			assert.NotContains(t, body, unavailableMessage)
			assert.NotContains(t, body, invalidGradeMessage)
			// A cached refusal would outlive the next alarm being switched on.
			assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		})
	}
	repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
}

// An expired alarm closes the page exactly like a switched-off one.
func TestFireListPageHandler_ExpiredAlarmIsForbidden(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	alarm := &entity.FireAlarmState{}
	require.NoError(t, alarm.Enable(time.Millisecond))
	controller := &FireListController{
		FireList: &firelistview.Service{Reports: repo, Alarm: alarm, Logger: zap.NewNop().Sugar()},
		Logger:   zap.NewNop().Sugar(),
	}

	time.Sleep(5 * time.Millisecond)

	w := performRequest(controller.FireListPageHandler(), "/firelist/5")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "inactive=true")
}

func TestFireListPageHandler_YearGroupAndFormGroup(t *testing.T) {
	repo, controller := newTestController()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.MatchedBy(
		func(f provider.VisitReportFilter) bool {
			return f.IsStudent != nil && *f.IsStudent &&
				len(f.YearGroups) == 1 && f.YearGroups[0] == 2 &&
				len(f.FormGroups) == 1 && f.FormGroups[0] == "2 B"
		})).Return(&provider.VisitReportResult{
		Total: 1,
		Rows:  []provider.VisitReportRow{{VisitorID: 3, Name: "Alice", Surname: "Adams", SignStatus: "signed_in"}},
	}, nil)

	// %20 in the path is decoded before it reaches the handler.
	w := performRequest(controller.FireListPageHandler(), "/firelist/2/2%20B")

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "grade=2 fg=2 B label=Grade 2 · 2 B")
	assert.Contains(t, body, "|3;Alice;Adams;signed_in;Signed In;0")
	repo.AssertExpectations(t)
}

func TestFireListPageHandler_StaffAndFormGroup(t *testing.T) {
	repo, controller := newTestController()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.MatchedBy(
		func(f provider.VisitReportFilter) bool {
			return f.IsStudent != nil && !*f.IsStudent && len(f.YearGroups) == 0 &&
				len(f.FormGroups) == 1 && f.FormGroups[0] == "MyGroup"
		})).Return(&provider.VisitReportResult{}, nil)

	w := performRequest(controller.FireListPageHandler(), "/firelist/staff/MyGroup")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "fg=MyGroup label=Staff · MyGroup")
	repo.AssertExpectations(t)
}

func TestFireListPageHandler_InvalidGradeWithFormGroup(t *testing.T) {
	repo, controller := newTestController()

	w := performRequest(controller.FireListPageHandler(), "/firelist/abc/2%20B")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "label=Fire list · 2 B")
	repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
}

func TestFireListPageHandler_ServiceErrorWithFormGroup(t *testing.T) {
	repo, controller := newTestController()

	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db down"))

	w := performRequest(controller.FireListPageHandler(), "/firelist/2/2%20B")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "label=Grade 2 · 2 B")
	assert.NotContains(t, w.Body.String(), "db down")
}

func TestFireListPageHandler_AlarmOffWithFormGroup(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	controller := &FireListController{
		FireList: &firelistview.Service{Reports: repo, Alarm: &entity.FireAlarmState{}, Logger: zap.NewNop().Sugar()},
		Logger:   zap.NewNop().Sugar(),
	}

	w := performRequest(controller.FireListPageHandler(), "/firelist/2/2%20B")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "fg=2 B label=Grade 2 · 2 B")
	assert.Contains(t, w.Body.String(), "inactive=true")
	repo.AssertNotCalled(t, "GetVisitReport", mock.Anything, mock.Anything, mock.Anything)
}
