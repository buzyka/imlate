package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
)

type Visitor struct {
	Connection *sql.DB `container:"type"`
}

func (r *Visitor) GetAll() ([]*entity.Visitor, error) {
	return r.FindAll()
}

func (r *Visitor) FindAll(opts ...provider.VisitorFilterOption) ([]*entity.Visitor, error) {
	cfg := &provider.VisitorFilterConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	query := "SELECT id, name, surname, email, is_student, grade, image, isams_id, isams_school_id, year_group, divisions, sync_hash, updated_at FROM visitors"
	args := make([]interface{}, 0)
	clauses := []string{"deleted_at IS NULL"}
	if len(cfg.ERPYearGroup) > 0 {
		placeholders := make([]string, len(cfg.ERPYearGroup))
		for i, yearGroup := range cfg.ERPYearGroup {
			placeholders[i] = "?"
			if yearGroup == nil {
				args = append(args, nil)
			} else {
				args = append(args, *yearGroup)
			}
		}
		clauses = append(clauses, fmt.Sprintf("year_group IN (%s)", strings.Join(placeholders, ", ")))
	}
	query += " WHERE " + strings.Join(clauses, " AND ")
	query += " ORDER BY id ASC"

	rows, err := r.Connection.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	visitors := []*entity.Visitor{}
	for rows.Next() {
		var tmpEmail sql.NullString
		var tmpGrade sql.NullInt32
		var tmpImage sql.NullString
		var tmpErpID sql.NullInt64
		var tmpErpSchoolID sql.NullString
		var tmpYearGroup sql.NullInt32
		var tmpDivisions sql.NullString
		var tmpUpdatedAt sql.NullTime
		var tmpSyncHash sql.NullString

		visitor := &entity.Visitor{}
		err := rows.Scan(
			&visitor.Id,
			&visitor.Name,
			&visitor.Surname,
			&tmpEmail,
			&visitor.IsStudent,
			&tmpGrade,
			&tmpImage,
			&tmpErpID,
			&tmpErpSchoolID,
			&tmpYearGroup,
			&tmpDivisions,
			&tmpSyncHash,
			&tmpUpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if tmpGrade.Valid {
			g := int(tmpGrade.Int32)
			visitor.Grade = &g
		}
		if tmpEmail.Valid {
			visitor.Email = tmpEmail.String
		}
		if tmpImage.Valid {
			visitor.Image = tmpImage.String
		} else {
			r.AddRandomImage(visitor)
		}
		if tmpErpID.Valid {
			visitor.ErpID = tmpErpID.Int64
		}
		if tmpErpSchoolID.Valid {
			visitor.ErpSchoolID = tmpErpSchoolID.String
		}
		if tmpYearGroup.Valid {
			visitor.ErpYearGroupID = tmpYearGroup.Int32
		}
		if tmpDivisions.Valid {
			err = json.Unmarshal([]byte(tmpDivisions.String), &visitor.ErpDivisions)
			if err != nil {
				visitor.ErpDivisions = []int32{}
			}
		}
		if tmpSyncHash.Valid {
			b, err := strconv.ParseUint(tmpSyncHash.String, 10, 64)
			if err == nil {
				visitor.SyncHash = b
			}
		}
		if tmpUpdatedAt.Valid {
			visitor.UpdatedAt = tmpUpdatedAt.Time
		}
		visitors = append(visitors, visitor)
	}
	return visitors, nil
}

func (r *Visitor) FindByKey(key string) (*entity.VisitDetails, error) {
	var tmpEmail sql.NullString
	var tmpGrade sql.NullInt32
	var tmpImage sql.NullString
	var tmpErpID sql.NullInt64
	var tmpErpSchoolID sql.NullString
	var tmpYearGroup sql.NullInt32
	var tmpDivisions sql.NullString
	var tmpSyncHash sql.NullString

	key = strings.ToUpper(key)
	row := r.Connection.QueryRow("SELECT v.id, v.name, v.surname, v.email, v.is_student, v.grade, v.image, v.isams_id, v.isams_school_id, v.year_group, v.divisions, v.sync_hash, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk ON vk.visitor_id = v.id WHERE vk.key_id = ? AND v.deleted_at IS NULL", key)

	visitor := &entity.Visitor{}
	visit := &entity.VisitDetails{
		Visitor: visitor,
	}

	err := row.Scan(
		&visitor.Id,
		&visitor.Name,
		&visitor.Surname,
		&tmpEmail,
		&visitor.IsStudent,
		&tmpGrade,
		&tmpImage,
		&tmpErpID,
		&tmpErpSchoolID,
		&tmpYearGroup,
		&tmpDivisions,
		&tmpSyncHash,
		&visit.Key,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.VisitDetails{}, nil
		}
		return nil, err
	}
	if tmpGrade.Valid {
		g := int(tmpGrade.Int32)
		visitor.Grade = &g
	}
	if tmpEmail.Valid {
		visitor.Email = tmpEmail.String
	}
	if tmpImage.Valid {
		visitor.Image = tmpImage.String
	} else {
		r.AddRandomImage(visitor)
	}
	if tmpErpID.Valid {
		visitor.ErpID = tmpErpID.Int64
	}
	if tmpErpSchoolID.Valid {
		visitor.ErpSchoolID = tmpErpSchoolID.String
	}
	if tmpYearGroup.Valid {
		visitor.ErpYearGroupID = tmpYearGroup.Int32
	}
	if tmpDivisions.Valid {
		err = json.Unmarshal([]byte(tmpDivisions.String), &visitor.ErpDivisions)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal divisions: %w", err)
		}
	}
	if tmpSyncHash.Valid {
		b, err := strconv.ParseUint(tmpSyncHash.String, 10, 64)
		if err == nil {
			visitor.SyncHash = b
		}
	}
	return visit, nil
}

func (r *Visitor) FindById(id int32) (*entity.Visitor, error) {
	var tmpEmail sql.NullString
	var tmpGrade sql.NullInt32
	var tmpImage sql.NullString
	var tmpErpID sql.NullInt64
	var tmpErpSchoolID sql.NullString
	var tmpYearGroup sql.NullInt32
	var tmpDivisions sql.NullString
	var tmpSyncHash sql.NullString

	row := r.Connection.QueryRow("SELECT id, name, surname, email, is_student, grade, image, isams_id, isams_school_id, year_group, divisions, sync_hash FROM visitors WHERE id = ? AND deleted_at IS NULL", id)
	student := &entity.Visitor{}
	err := row.Scan(
		&student.Id,
		&student.Name,
		&student.Surname,
		&tmpEmail,
		&student.IsStudent,
		&tmpGrade,
		&tmpImage,
		&tmpErpID,
		&tmpErpSchoolID,
		&tmpYearGroup,
		&tmpDivisions,
		&tmpSyncHash,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.Visitor{}, nil
		}
		return nil, err
	}
	if tmpGrade.Valid {
		g := int(tmpGrade.Int32)
		student.Grade = &g
	}
	if tmpEmail.Valid {
		student.Email = tmpEmail.String
	}
	if tmpImage.Valid {
		student.Image = tmpImage.String
	} else {
		r.AddRandomImage(student)
	}
	if tmpErpID.Valid {
		student.ErpID = tmpErpID.Int64
	}
	if tmpErpSchoolID.Valid {
		student.ErpSchoolID = tmpErpSchoolID.String
	}
	if tmpYearGroup.Valid {
		student.ErpYearGroupID = tmpYearGroup.Int32
	}
	if tmpDivisions.Valid {
		err = json.Unmarshal([]byte(tmpDivisions.String), &student.ErpDivisions)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal divisions: %w", err)
		}
	}
	if tmpSyncHash.Valid {
		b, err := strconv.ParseUint(tmpSyncHash.String, 10, 64)
		if err == nil {
			student.SyncHash = b
		}
	}
	return student, nil
}

func (r *Visitor) AddKeyToVisitor(visitor *entity.Visitor, key string) error {
	details, err := r.FindByKey(key)
	if err != nil {
		return fmt.Errorf("search by key error: %s", err.Error())
	}

	if details.Visitor != nil && details.Visitor.Id > 0 {
		if details.Visitor.Id == visitor.Id {
			return nil
		}
		return fmt.Errorf("key already assigned to another visitor")
	}

	_, err = r.Connection.Exec("INSERT INTO visitor_key (visitor_id, key_id) VALUES (?, ?)", visitor.Id, key)
	if err != nil {
		return err
	}
	return nil
}

func (r *Visitor) SaveVisitor(visitor *entity.Visitor) error {
	if visitor.Id > 0 {
		return r.updateVisitor(visitor)
	} else {
		return r.insertVisitor(visitor)
	}
}

func (r *Visitor) AddVisitor(visitor *entity.Visitor) error {
	return r.SaveVisitor(visitor)
}

func (r *Visitor) updateVisitor(visitor *entity.Visitor) error {
	var id sql.NullInt32
	if visitor.Id > 0 {
		id = sql.NullInt32{
			Int32: visitor.Id,
			Valid: true,
		}
	}

	var grade sql.NullInt32
	if visitor.Grade != nil {
		grade = sql.NullInt32{Int32: int32(*visitor.Grade), Valid: true}
	}

	var email sql.NullString
	if visitor.Email != "" {
		email = sql.NullString{String: visitor.Email, Valid: true}
	}

	var erpID sql.NullInt64
	if visitor.ErpID != 0 {
		erpID = sql.NullInt64{
			Int64: int64(visitor.ErpID),
			Valid: true,
		}
	}

	var erpSchoolID sql.NullString
	if visitor.ErpSchoolID != "" {
		erpSchoolID = sql.NullString{
			String: visitor.ErpSchoolID,
			Valid:  true,
		}
	}

	divisionsStr, err := json.Marshal(visitor.ErpDivisions)
	if err != nil {
		return err
	}

	_, err = r.Connection.Exec(
		"UPDATE visitors SET name = ?, surname = ?, email = ?, is_student = ?, grade = ?, image = ?, isams_id = ?, isams_school_id = ?, year_group = ?, divisions = ?, updated_at = ?, sync_hash = ? WHERE id = ?",
		visitor.Name,
		visitor.Surname,
		email,
		visitor.IsStudent,
		grade,
		visitor.Image,
		erpID,
		erpSchoolID,
		visitor.ErpYearGroupID,
		string(divisionsStr),
		visitor.UpdatedAt,
		fmt.Sprintf("%d", visitor.GetSyncHash()),
		id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *Visitor) insertVisitor(visitor *entity.Visitor) error {
	var grade sql.NullInt32
	if visitor.Grade != nil {
		grade = sql.NullInt32{Int32: int32(*visitor.Grade), Valid: true}
	}

	var email sql.NullString
	if visitor.Email != "" {
		email = sql.NullString{String: visitor.Email, Valid: true}
	}

	var erpID sql.NullInt64
	if visitor.ErpID != 0 {
		erpID = sql.NullInt64{
			Int64: int64(visitor.ErpID),
			Valid: true,
		}
	}

	var erpSchoolID sql.NullString
	if visitor.ErpSchoolID != "" {
		erpSchoolID = sql.NullString{
			String: visitor.ErpSchoolID,
			Valid:  true,
		}
	}

	divisionsStr, err := json.Marshal(visitor.ErpDivisions)
	if err != nil {
		return err
	}

	result, err := r.Connection.Exec(
		"INSERT INTO visitors (name, surname, email, is_student, grade, image, isams_id, isams_school_id, year_group, divisions, updated_at, sync_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		visitor.Name,
		visitor.Surname,
		email,
		visitor.IsStudent,
		grade,
		visitor.Image,
		erpID,
		erpSchoolID,
		visitor.ErpYearGroupID,
		string(divisionsStr),
		visitor.UpdatedAt,
		fmt.Sprintf("%d", visitor.GetSyncHash()),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("visitor has not been saved correctly: %s", err.Error())
	}
	visitor.Id = int32(id)
	return nil
}

func (r *Visitor) DeleteVisitor(id int32) error {
	result, err := r.Connection.Exec("UPDATE visitors SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("visitor not found or already deleted")
	}
	return nil
}

func (r *Visitor) RemoveKeyFromVisitor(visitorID int32, key string) error {
	key = strings.ToUpper(key)
	result, err := r.Connection.Exec("DELETE FROM visitor_key WHERE visitor_id = ? AND key_id = ?", visitorID, key)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("key not found for this visitor")
	}
	return nil
}

func (r *Visitor) FindKeysByVisitorId(visitorID int32) ([]string, error) {
	rows, err := r.Connection.Query("SELECT key_id FROM visitor_key WHERE visitor_id = ? ORDER BY created_at ASC", visitorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *Visitor) UpdateVisitorImage(id int32, imagePath string) error {
	result, err := r.Connection.Exec("UPDATE visitors SET image = ?, updated_at = NOW() WHERE id = ? AND deleted_at IS NULL", imagePath, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("visitor not found")
	}
	return nil
}

func (r *Visitor) AddRandomImage(student *entity.Visitor) {
	source := rand.NewSource(time.Now().UnixNano())
	rmd := rand.New(source)
	fName := rmd.Intn(11) + 1
	student.Image = fmt.Sprintf("/assets/img/teachers/%d.jpg", fName)
}
