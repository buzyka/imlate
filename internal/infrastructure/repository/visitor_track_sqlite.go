package repository

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buzyka/imlate/internal/isb/entity"
)

type VisitorTrack struct {
	Connection *sql.DB `container:"type"`
}

func (r *VisitorTrack) Store(vt *entity.VisitTrack) (*entity.VisitTrack, error) {
	res, err := r.Connection.Exec(
		"INSERT INTO track (visitor_id, sign_in) VALUES (?, ?)",
		vt.VisitorId,
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
	vt.Id = vt1.Id
	vt.CreatedAt = vt1.CreatedAt
	r.writeToTheFile(vt)
	return vt, err1
}

func (r *VisitorTrack) GetById(id int64) (*entity.VisitTrack, error) {
	row := r.Connection.QueryRow("SELECT t.id, t.visitor_id, t.sign_in, t.created_at FROM track AS t WHERE id = ?", id)
	track := &entity.VisitTrack{}
	err := row.Scan(
		&track.Id,
		&track.VisitorId,
		&track.SignedIn,
		&track.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return track, nil
}

func (r *VisitorTrack) writeToTheFile(vt *entity.VisitTrack) {
	rootPath, err := getRootPath()
	if err != nil {
		return
	}
	
	// The path to the CSV file
    filePath := rootPath + "/output/data.csv"

    // The new row to be added
    newRow := []string{vt.CreatedAt.Format("2006-01-02 15:04:05"), vt.VisitorId, vt.Visitor.Name, vt.Visitor.Surname}

    // Open the file in append mode or create it if it doesn't exist
    file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        panic(fmt.Sprint("failed to open file: %s", err))
    }
    defer file.Close()

    // Create a new CSV writer
    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Write the new row to the CSV file
    if err := writer.Write(newRow); err != nil {
        panic(fmt.Sprint("failed to write to file: %s", err))
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