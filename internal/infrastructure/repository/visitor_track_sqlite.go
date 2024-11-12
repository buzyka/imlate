package repository

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/buzyka/imlate/internal/isb/stat"
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
		return nil, err
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

func (r *VisitorTrack) GetDailyStat(date time.Time) (*stat.DailyGeneral, error) {
	type DailyStat struct {
		TrackDate   time.Time
		VisitorCount int
	}

	var visitorCount int
	err := r.Connection.QueryRow("SELECT COUNT(*) FROM visitors WHERE is_student = 0").Scan(&visitorCount)
	if err != nil {
        return nil, err
    }

	query := `
        SELECT 
            DATE(t.created_at) AS track_date,
            COUNT(DISTINCT t.visitor_id) AS tracked_visitor_count
        FROM track t 
        WHERE t.created_at >= ? 
        GROUP BY track_date
        ORDER BY track_date;
    `
	sevenDaysAgo := date.AddDate(0, 0, -8)
    rows, err := r.Connection.Query(query, sevenDaysAgo.Format("2006-01-02"))
    if err != nil {
        return nil, err
    }
    defer rows.Close()


	stats := make(map[string]int)
    for rows.Next() {
		var trackDateStr string 
		var count int
        err := rows.Scan(&trackDateStr, &count)
        if err != nil {
            return nil, err
        }
		stats[trackDateStr] = count
    }

	today, ok := stats[date.Format("2006-01-02")];
	if !ok {
		today = 0
	}
	yesterday, ok := stats[date.AddDate(0, 0, -1).Format("2006-01-02")];
	if !ok {
		yesterday = 0
	}

	result := &stat.DailyGeneral{
		TotalVisitors: visitorCount,
		RegisteredVisitors: today,
	}
 
	result.PercentComparingToYesterdayTrend, result.PercentComparingToYesterday = compareValues(today, yesterday)

	var days int;
	var sum int;
	for i := 8; i > 1; i-- {
		ds, ok := stats[date.AddDate(0, 0, -i).Format("2006-01-02")];
		if ok {
			days++
			sum += ds
		}
	}
	if days > 0 {
		average := int(math.Round(float64(sum) / float64(days)))
		result.PercentComparingToLastWeekTrend, result.PercentComparingToLastWeek = compareValues(today, average)
	} else {
		result.PercentComparingToLastWeek = 0
		result.PercentComparingToLastWeekTrend = stat.TrendEQUAL
	}	

	return result, nil

	// return &stat.DailyGeneral{
	// 	TotalVisitors: 10,
	// 	RegisteredVisitors: 5,
	// 	PercentComparingToYesterday: 25.5,
	// 	PercentComparingToYesterdayTrend: stat.TrendDOWN,
	// 	PercentComparingToLastWeek: 15.2678,
	// 	PercentComparingToLastWeekTrend: stat.TrendEQUAL,
	// }, nil
}

func compareValues(a, b int) (trend string, percent float64) {	
	if a > b {
		return stat.TrendUP, float64(a - b) / float64(a) * 100
	}
	if a < b {
		return stat.TrendDOWN, float64(b - a) / float64(a) * 100
	}
	return stat.TrendEQUAL, 0
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
    defer file.Close()

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