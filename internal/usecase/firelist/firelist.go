package firelist

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/buzyka/imlate/internal/version"
	"go.uber.org/zap"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	// maxFireListRows is deliberately larger than any realistic class. The whole
	// roster must be on the page: a teacher paging through results while
	// evacuating is not an acceptable interaction.
	maxFireListRows = 2000

	// staffGradeParam selects everyone who is not a student.
	staffGradeParam = "staff"
	// staffGradeLabel heads that group. Everyone in this bucket is an adult who
	// works here, so the roster names them rather than filing them under a
	// leftovers heading like "Other".
	staffGradeLabel = "Staff"
	// fallbackGradeLabel heads the error page when the URL segment names no
	// recognisable class.
	fallbackGradeLabel = "Fire list"

	dayLayout       = "2006-01-02"
	clockLayout     = "15:04"
	statusSignedIn  = "signed_in"
	statusSignedOut = "signed_out"
	statusNotSigned = "not_signed"
)

// ErrInvalidGrade is returned when the URL segment is neither "staff" nor an
// integer year group. Mirrors adminapi.ErrInvalidRequestFormat in style so
// controllers can branch on it with errors.Is.
var ErrInvalidGrade = errors.New("invalid grade")

// ErrAlarmInactive is returned when no fire alarm is running. The roster names
// every child in a class and says who is in the building, so it is readable
// only for the duration of an alarm an admin has switched on.
var ErrAlarmInactive = errors.New("fire alarm is not active")

// FireListPageData is the template data for firelist.html.
type FireListPageData struct {
	GradeParam string // "5" | "staff" — raw URL value, used as the localStorage key
	// FormGroupParam narrows the list to one class ("2 B"); empty for the whole
	// year group or all staff. Part of the localStorage key.
	FormGroupParam string
	GradeLabel     string // "Grade 5" | "Staff" | "Grade 2 · 2 B" — page heading
	Day            string // "2026-09-15" — localStorage key part + header caption
	GeneratedAt    string // "09:41" — when the data snapshot was taken
	AppVersion     string // for ?v= cache busting on CSS/JS
	Rows           []FireListRow
	Counts         FireListCounts
	Error          string // non-empty => template renders an error banner instead of the table
	// AlarmInactive => template renders a neutral "no alarm running" notice
	// instead of the table. Distinct from Error: this is an ordinary state, not
	// a failure, and must not be dressed up as one.
	AlarmInactive bool
}

type FireListRow struct {
	VisitorID   int32
	Name        string
	Surname     string
	Status      string // "signed_in" | "signed_out" | "not_signed" — CSS class suffix
	StatusLabel string // "Signed In" | "Signed Out" | "No Status"
	StatusRank  int    // 0 | 1 | 2 — data-rank attribute for client-side sorting
}

type FireListCounts struct {
	SignedIn  int
	SignedOut int
	NoStatus  int
	Total     int
}

// Service assembles the evacuation roster for one class.
type Service struct {
	Reports provider.VisitDailyReportRepository `container:"type"`
	Alarm   *entity.FireAlarmState              `container:"type"`
	Logger  *zap.SugaredLogger                  `container:"type"`
}

// GetFireList returns the roster for the given URL grade segment, for today,
// optionally narrowed to one form group. A blank form group means no narrowing.
func (s *Service) GetFireList(gradeParam, formGroupParam string) (FireListPageData, error) {
	// Checked first, before the grade is even parsed and before anything
	// touches the database. The page is public, so this gate is the only thing
	// standing between an anonymous request and the names of every child in a
	// class. Keeping it in the service rather than the controller means a
	// future second caller cannot skip it by accident.
	if !s.Alarm.IsActive() {
		return FireListPageData{}, ErrAlarmInactive
	}

	filter, label, err := parseGrade(gradeParam)
	if err != nil {
		return FireListPageData{}, err
	}
	formGroup := normalizeFormGroupParam(formGroupParam)
	if formGroup != "" {
		filter.FormGroups = []string{formGroup}
		label = withFormGroup(label, formGroup)
	}

	day := util.Now()

	// The repository filters `r.day >= from AND r.day < to`, so a half-open
	// [day, day+1) range is exactly one calendar day. Do NOT route this through
	// adminapi.GetReportsVisits: that helper adds an extra day to `to`
	// internally, so it returns two days of rows — which is why the admin SPA
	// has to dedupe by visitor client-side. Here a duplicated child on an
	// evacuation roster would be counted twice.
	from := day
	to := day.AddDate(0, 0, 1)

	// Before the day's first card scan the report table holds no rows for today,
	// so without this the teacher would be shown an empty class. The call
	// short-circuits on a single `SELECT 1 ... LIMIT 1` once any row exists, so
	// the full insert runs at most once per day.
	if err := s.Reports.EnsureDayRows(day); err != nil {
		s.logf("firelist: EnsureDayRows failed for %s: %v", day.Format(dayLayout), err)
		// Deliberately not fatal: stale or partial rows still beat no page.
	}

	filter.Page = 1
	filter.PageSize = maxFireListRows
	// OrderField stays empty: the priority ordering below is not expressible as
	// a single SQL sort, and the result set is one class at most.

	result, err := s.Reports.GetVisitReport(from, to, filter)
	if err != nil {
		return FireListPageData{}, fmt.Errorf("failed to read fire list report: %w", err)
	}

	rows := make([]FireListRow, 0, len(result.Rows))
	counts := FireListCounts{}
	for _, r := range result.Rows {
		status, statusLabel, rank := classifyStatus(r.SignStatus)
		switch rank {
		case 0:
			counts.SignedIn++
		case 1:
			counts.SignedOut++
		default:
			counts.NoStatus++
		}
		rows = append(rows, FireListRow{
			VisitorID:   r.VisitorID,
			Name:        r.Name,
			Surname:     r.Surname,
			Status:      status,
			StatusLabel: statusLabel,
			StatusRank:  rank,
		})
	}
	counts.Total = len(rows)

	sortRows(rows)

	return FireListPageData{
		GradeParam:     gradeParam,
		FormGroupParam: formGroup,
		GradeLabel:     label,
		Day:            day.Format(dayLayout),
		GeneratedAt:    day.Format(clockLayout),
		AppVersion:     version.Version,
		Rows:           rows,
		Counts:         counts,
	}, nil
}

// ErrorPageData builds page data that renders only the error banner, so the
// controller can serve a readable page instead of a JSON blob to someone who is
// mid-evacuation.
func ErrorPageData(gradeParam, formGroupParam, message string) FireListPageData {
	day := util.Now()
	formGroup := normalizeFormGroupParam(formGroupParam)
	return FireListPageData{
		GradeParam:     gradeParam,
		FormGroupParam: formGroup,
		GradeLabel:     withFormGroup(gradeLabel(gradeParam), formGroup),
		Day:            day.Format(dayLayout),
		GeneratedAt:    day.Format(clockLayout),
		AppVersion:     version.Version,
		Error:          message,
	}
}

// InactivePageData builds page data for the "no alarm running" page. It
// carries no rows and no counts by construction, so the gate cannot leak a
// roster through the page it renders when it refuses one.
func InactivePageData(gradeParam, formGroupParam string) FireListPageData {
	day := util.Now()
	formGroup := normalizeFormGroupParam(formGroupParam)
	return FireListPageData{
		GradeParam:     gradeParam,
		FormGroupParam: formGroup,
		GradeLabel:     withFormGroup(gradeLabel(gradeParam), formGroup),
		Day:            day.Format(dayLayout),
		GeneratedAt:    day.Format(clockLayout),
		AppVersion:     version.Version,
		AlarmInactive:  true,
	}
}

// parseGrade maps the URL segment onto a repository filter and a heading.
func parseGrade(gradeParam string) (provider.VisitReportFilter, string, error) {
	if strings.EqualFold(gradeParam, staffGradeParam) {
		isStudent := false
		return provider.VisitReportFilter{IsStudent: &isStudent}, staffGradeLabel, nil
	}

	n, err := strconv.Atoi(gradeParam)
	if err != nil {
		return provider.VisitReportFilter{}, "", fmt.Errorf("%w: %q is not a year group", ErrInvalidGrade, gradeParam)
	}

	isStudent := true
	return provider.VisitReportFilter{
		IsStudent:  &isStudent,
		YearGroups: []int{n},
	}, fmt.Sprintf("Grade %d", n), nil
}

// gradeLabel is the best-effort heading for a param that may not be valid. An
// unparseable param still needs a heading: it is only ever reached from the
// error page, and an empty <h1> plus a title reading " — Evacuation" tells a
// teacher nothing about what they just opened.
func gradeLabel(gradeParam string) string {
	if strings.EqualFold(gradeParam, staffGradeParam) {
		return staffGradeLabel
	}
	if n, err := strconv.Atoi(gradeParam); err == nil {
		return fmt.Sprintf("Grade %d", n)
	}
	return fallbackGradeLabel
}

// normalizeFormGroupParam applies the stored-value rules to the URL segment, so
// " 2 B " finds "2 B" and a blank segment means "whole year group".
func normalizeFormGroupParam(formGroupParam string) string {
	if formGroup := entity.NormalizeFormGroup(&formGroupParam); formGroup != nil {
		return *formGroup
	}
	return ""
}

// withFormGroup appends the class to a heading: "Grade 2" -> "Grade 2 · 2 B".
func withFormGroup(label, formGroup string) string {
	if formGroup == "" {
		return label
	}
	return label + " · " + formGroup
}

// classifyStatus turns the repository's sign_status into the CSS suffix, the
// human label and the evacuation priority rank. Anything unrecognised is
// treated as "not seen today", which is the safe default: it keeps the person
// visible at the top of the unaccounted-for group rather than hiding them.
func classifyStatus(signStatus string) (string, string, int) {
	switch signStatus {
	case statusSignedIn:
		return statusSignedIn, "Signed In", 0
	case statusSignedOut:
		return statusSignedOut, "Signed Out", 1
	default:
		return statusNotSigned, "No Status", 2
	}
}

// sortRows orders by evacuation priority, then alphabetically. A plain
// lowercase byte compare would push names like "Muñecas" and "Thézé" past "Z",
// which breaks a teacher scanning the list top to bottom — so surnames and
// names are compared with a Unicode collator that ignores case and diacritics.
func sortRows(rows []FireListRow) {
	// A Collator keeps internal buffers and is not safe for concurrent use, so
	// it is built per call rather than cached on the Service singleton.
	c := collate.New(language.Und, collate.Loose)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].StatusRank != rows[j].StatusRank {
			return rows[i].StatusRank < rows[j].StatusRank
		}
		if cmp := c.CompareString(rows[i].Surname, rows[j].Surname); cmp != 0 {
			return cmp < 0
		}
		return c.CompareString(rows[i].Name, rows[j].Name) < 0
	})
}

func (s *Service) logf(format string, args ...interface{}) {
	if s.Logger == nil {
		return
	}
	s.Logger.Errorf(format, args...)
}
