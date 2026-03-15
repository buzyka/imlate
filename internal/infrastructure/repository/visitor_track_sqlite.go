package repository

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
)

type VisitorTrack struct {
	Connection *sql.DB `container:"type"`
}

func (r *VisitorTrack) Store(vt *entity.VisitTrack) (*entity.VisitTrack, error) {
	res, err := r.Connection.Exec(
		"INSERT INTO track (visitor_id, key_id, sign_in, created_at) VALUES (?, ?, ?,  NOW())",
		vt.VisitorId,
		vt.VisitKey,
		vt.SignedIn,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	vt1, err1 := r.GetById(id)
	if err1 != nil {
		return nil, err1
	}
	vt.Id = vt1.Id
	vt.VisitKey = vt1.VisitKey
	vt.CreatedAt = vt1.CreatedAt
	r.writeToTheFile(vt)
	return vt, err1
}

func (r *VisitorTrack) GetById(id int64) (*entity.VisitTrack, error) {
	var createdAtRaw []byte

	row := r.Connection.QueryRow("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?", id)
	track := &entity.VisitTrack{}
	err := row.Scan(
		&track.Id,
		&track.VisitorId,
		&track.VisitKey,
		&track.SignedIn,
		&createdAtRaw,
	)
	if err != nil {
		return nil, err
	}

	track.CreatedAt, err = time.Parse("2006-01-02 15:04:05", string(createdAtRaw))
	if err != nil {
		track.CreatedAt, err = time.Parse(time.RFC3339, string(createdAtRaw))
		if err != nil {
			return nil, err
		}
	}
	return track, nil
}

func (r *VisitorTrack) CountEventsByVisitorIdSince(visitorId int32, date time.Time) (int, error) {
	var count int
	err := r.Connection.QueryRow(
		"SELECT COUNT(*) FROM track WHERE visitor_id = ? AND created_at > ?",
		visitorId,
		date,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *VisitorTrack) GetVisitReport(from, to time.Time, filter provider.VisitReportFilter) (*provider.VisitReportResult, error) {
	visitorWhere, visitorArgs := r.buildVisitorFilters(filter)

	baseQuery := r.buildReportBaseQuery(visitorWhere)

	havingClause := r.buildHavingClause(filter)

	countQuery := "SELECT COUNT(*) FROM (" + baseQuery + havingClause + ") AS cnt"
	countArgs := append([]interface{}{from, to}, visitorArgs...)

	var total int
	if err := r.Connection.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	dataQuery := baseQuery + havingClause + " ORDER BY vd.visit_date, v.surname LIMIT ? OFFSET ?"
	offset := (filter.Page - 1) * filter.PageSize
	dataArgs := append([]interface{}{from, to}, visitorArgs...)
	dataArgs = append(dataArgs, filter.PageSize, offset)

	rows, err := r.Connection.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("data query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []provider.VisitReportRow
	for rows.Next() {
		var row provider.VisitReportRow
		var visitDateStr string
		var signedInRaw, signedOutRaw sql.NullString
		var durationMinutes sql.NullInt64
		var yearGroup sql.NullInt64

		if err := rows.Scan(
			&visitDateStr,
			&row.VisitorID,
			&row.Name,
			&row.Surname,
			&row.IsStudent,
			&yearGroup,
			&row.VisitsCount,
			&row.SignStatus,
			&signedInRaw,
			&signedOutRaw,
			&durationMinutes,
		); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		visitDate, err := r.parseDate(visitDateStr)
		if err != nil {
			return nil, fmt.Errorf("visit_date parse failed: %w", err)
		}
		row.VisitDate = visitDate

		if yearGroup.Valid {
			yg := int(yearGroup.Int64)
			row.YearGroup = &yg
		}
		if signedInRaw.Valid {
			t, err := r.parseTimestamp(signedInRaw.String)
			if err != nil {
				return nil, fmt.Errorf("signed_in parse failed: %w", err)
			}
			row.SignedIn = &t
		}
		if signedOutRaw.Valid {
			t, err := r.parseTimestamp(signedOutRaw.String)
			if err != nil {
				return nil, fmt.Errorf("signed_out parse failed: %w", err)
			}
			row.SignedOut = &t
		}
		if durationMinutes.Valid {
			d := int(durationMinutes.Int64)
			row.DurationMinutes = &d
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

func (r *VisitorTrack) buildReportBaseQuery(visitorWhere string) string {
	return `SELECT
    vd.visit_date,
    v.id AS visitor_id,
    v.name,
    v.surname,
    v.is_student,
    v.year_group,
    COUNT(t.id) AS visits_count,
    CASE
        WHEN COUNT(t.id) = 0 THEN 'not_signed'
        WHEN MOD(COUNT(t.id), 2) = 0 THEN 'signed_out'
        ELSE 'signed_in'
    END AS sign_status,
    MIN(t.created_at) AS signed_in,
    MAX(t.created_at) AS signed_out,
    CASE
        WHEN COUNT(t.id) > 1
        THEN TIMESTAMPDIFF(MINUTE, MIN(t.created_at), MAX(t.created_at))
        ELSE NULL
    END AS duration_minutes
FROM
(
    SELECT
        DATE(t.created_at) AS visit_date,
        DATE(t.created_at) AS start_date,
        DATE_ADD(DATE(t.created_at), INTERVAL 1 DAY) AS end_date
    FROM track t
    WHERE t.created_at >= ? AND t.created_at < ?
    GROUP BY DATE(t.created_at)
) vd
CROSS JOIN visitors v
LEFT JOIN track t
    ON t.visitor_id = v.id
    AND t.created_at >= vd.start_date
    AND t.created_at < vd.end_date
WHERE v.deleted_at IS NULL` + visitorWhere + `
GROUP BY
    vd.visit_date,
    v.id,
    v.name,
    v.surname,
    v.is_student,
    v.year_group`
}

func (r *VisitorTrack) buildVisitorFilters(filter provider.VisitReportFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if filter.IsStudent != nil {
		clauses = append(clauses, "v.is_student = ?")
		args = append(args, *filter.IsStudent)
	}
	if filter.YearGroup != nil {
		clauses = append(clauses, "v.year_group = ?")
		args = append(args, *filter.YearGroup)
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

func (r *VisitorTrack) buildHavingClause(filter provider.VisitReportFilter) string {
	if filter.SignStatus == nil {
		return ""
	}

	switch *filter.SignStatus {
	case "not_signed":
		return " HAVING COUNT(t.id) = 0"
	case "signed_in":
		return " HAVING MOD(COUNT(t.id), 2) = 1"
	case "signed_out":
		return " HAVING MOD(COUNT(t.id), 2) = 0 AND COUNT(t.id) > 0"
	default:
		return ""
	}
}

func (r *VisitorTrack) parseDate(raw string) (time.Time, error) {
	if len(raw) >= 10 {
		t, err := time.Parse("2006-01-02", raw[:10])
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %s", raw)
}

func (r *VisitorTrack) parseTimestamp(raw string) (time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05", raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, err
		}
	}
	return t, nil
}

func (r *VisitorTrack) writeToTheFile(vt *entity.VisitTrack) {
	rootPath, err := getRootPath()
	if err != nil {
		return
	}

	// The path to the CSV file
	filePath := rootPath + "/output/data.csv"

	// The new row to be added
	newRow := []string{vt.CreatedAt.Format("2006-01-02 15:04:05"), strconv.Itoa(int(vt.Visitor.Id)), vt.Visitor.Name, vt.Visitor.Surname}

	// Open the file in append mode or create it if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("failed to open file: %s", err))
	}
	defer func() { _ = file.Close() }()

	// Create a new CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the new row to the CSV file
	if err := writer.Write(newRow); err != nil {
		panic(fmt.Sprintf("failed to write to file: %s", err))
	}
}

func getRootPath() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		for {
			if info, errDir := os.Stat(cwd + "/output"); errDir == nil && info.IsDir() {
				return cwd, nil
			}
			parent := filepath.Dir(cwd)
			if parent == cwd {
				break
			}
			cwd = parent
		}
	}
	return "", os.ErrNotExist
}
