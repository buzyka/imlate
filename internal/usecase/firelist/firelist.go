package firelist

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/buzyka/imlate/internal/version"
	"go.uber.org/zap"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	// maxFireListRows is deliberately larger than any realistic class.
	maxFireListRows = 2000

	// staffGradeParam selects everyone who is not a student.
	staffGradeParam = "staff"
	staffGradeLabel = "Staff"
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

// FireListPageData is the template data for firelist.html.
type FireListPageData struct {
	GradeParam  string // "5" | "staff" — raw URL value, used as the localStorage key
	GradeLabel  string // "Grade 5" | "Staff" — page heading
	Day         string // "2026-09-15" — localStorage key part + header caption
	GeneratedAt string // "09:41" — when the data snapshot was taken
	AppVersion  string // for ?v= cache busting on CSS/JS
	Rows        []FireListRow
	Counts      FireListCounts
	Error       string // non-empty => template renders an error banner instead of the table
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
	Logger  *zap.SugaredLogger                  `container:"type"`
}

// GetFireList returns the roster for the given URL grade segment, for today.
func (s *Service) GetFireList(gradeParam string) (FireListPageData, error) {
	filter, label, err := parseGrade(gradeParam)
	if err != nil {
		return FireListPageData{}, err
	}

	day := util.Now()
	from := day
	to := day.AddDate(0, 0, 1)

	if err := s.Reports.EnsureDayRows(day); err != nil {
		s.logf("firelist: EnsureDayRows failed for %s: %v", day.Format(dayLayout), err)
	}

	filter.Page = 1
	filter.PageSize = maxFireListRows

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
		GradeParam:  gradeParam,
		GradeLabel:  label,
		Day:         day.Format(dayLayout),
		GeneratedAt: day.Format(clockLayout),
		AppVersion:  version.Version,
		Rows:        rows,
		Counts:      counts,
	}, nil
}

// ErrorPageData builds page data that renders only the error banner, so the
// controller can serve a readable page instead of a JSON blob to someone who is
// mid-evacuation.
func ErrorPageData(gradeParam, message string) FireListPageData {
	day := util.Now()
	return FireListPageData{
		GradeParam:  gradeParam,
		GradeLabel:  gradeLabel(gradeParam),
		Day:         day.Format(dayLayout),
		GeneratedAt: day.Format(clockLayout),
		AppVersion:  version.Version,
		Error:       message,
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
