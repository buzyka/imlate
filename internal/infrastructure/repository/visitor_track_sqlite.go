package repository

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/buzyka/imlate/internal/isb/entity"
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
