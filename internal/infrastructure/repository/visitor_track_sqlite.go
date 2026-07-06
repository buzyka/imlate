package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/google/uuid"
)

type VisitorTrack struct {
	Connection *sql.DB `container:"type"`
}

func (r *VisitorTrack) Store(vt *entity.VisitTrack) (*entity.VisitTrack, error) {
	var adminIDStr *string
	if vt.AdminID != nil {
		s := vt.AdminID.String()
		adminIDStr = &s
	}
	res, err := r.Connection.Exec(
		"INSERT INTO track (visitor_id, key_id, sign_in, admin_id, description, created_at) VALUES (?, ?, ?, ?, ?, NOW())",
		vt.VisitorId,
		vt.VisitKey,
		vt.SignedIn,
		adminIDStr,
		vt.Description,
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
	return vt, err1
}

func (r *VisitorTrack) GetById(id int64) (*entity.VisitTrack, error) {
	var createdAtRaw []byte
	var keyIDRaw sql.NullString
	var adminIDRaw sql.NullString
	var descriptionRaw sql.NullString

	row := r.Connection.QueryRow("SELECT t.id, t.visitor_id, t.key_id, t.sign_in, t.admin_id, t.description, t.created_at FROM track AS t WHERE id = ?", id)
	track := &entity.VisitTrack{}
	err := row.Scan(
		&track.Id,
		&track.VisitorId,
		&keyIDRaw,
		&track.SignedIn,
		&adminIDRaw,
		&descriptionRaw,
		&createdAtRaw,
	)
	if err != nil {
		return nil, err
	}

	if keyIDRaw.Valid {
		track.VisitKey = &keyIDRaw.String
	}
	if adminIDRaw.Valid {
		parsed, err := uuid.Parse(adminIDRaw.String)
		if err != nil {
			return nil, fmt.Errorf("invalid admin_id: %w", err)
		}
		track.AdminID = &parsed
	}
	if descriptionRaw.Valid {
		track.Description = &descriptionRaw.String
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
