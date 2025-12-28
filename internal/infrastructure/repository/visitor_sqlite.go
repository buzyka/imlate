package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/isb/entity"
)

type Visitor struct {
	Connection *sql.DB `container:"type"`
}

func (r *Visitor) GetAll() ([]*entity.Visitor, error) {
	rows, err := r.Connection.Query("SELECT id, name, surname, is_student, grade, image, isams_id, isams_school_id, updated_at FROM visitors ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	visitors := []*entity.Visitor{}
	for rows.Next() {
		var tmpGrade sql.NullInt32
		var tmpImage sql.NullString
		var tmpErpID sql.NullInt64
		var tmpErpSchoolID sql.NullString
		var tmpUpdatedAt sql.NullTime

		visitor := &entity.Visitor{}
		err := rows.Scan(
			&visitor.Id,
			&visitor.Name,
			&visitor.Surname,
			&visitor.IsStudent,
			&tmpGrade,
			&tmpImage,
			&tmpErpID,
			&tmpErpSchoolID,
			&tmpUpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if tmpGrade.Valid {
			visitor.Grade = int(tmpGrade.Int32)
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
		if tmpUpdatedAt.Valid {
			visitor.UpdatedAt = tmpUpdatedAt.Time
		}
		visitors = append(visitors, visitor)
	}
	return visitors, nil
}

func (r *Visitor) FindByKey(key string) (*entity.VisitDetails, error) {
	var tmpGrade sql.NullInt32
	var tmpImage sql.NullString
	var tmpErpID sql.NullInt64
	var tmpErpSchoolID sql.NullString
	var tmpYearGroup sql.NullInt32
	var tmpDivisions sql.NullString

	key = strings.ToUpper(key)
	row := r.Connection.QueryRow("SELECT v.id, v.name, v.surname, v.is_student, v.grade, v.image, v.isams_id, v.isams_school_id, v.year_group, v.divisions, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk ON vk.visitor_id = v.id WHERE vk.key_id = ?", key)

	visitor := &entity.Visitor{}
	visit := &entity.VisitDetails{
		Visitor: visitor,
	}

	err := row.Scan(
		&visitor.Id,
		&visitor.Name,
		&visitor.Surname,
		&visitor.IsStudent,
		&tmpGrade,
		&tmpImage,
		&tmpErpID,
		&tmpErpSchoolID,
		&tmpYearGroup,
		&tmpDivisions,
		&visit.Key,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.VisitDetails{}, nil
		}
		return nil, err
	}
	if tmpGrade.Valid {
		visitor.Grade = int(tmpGrade.Int32)
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
	return visit, nil
}

func (r *Visitor) FindById(id int32) (*entity.Visitor, error) {
	var tmpGrade sql.NullInt32
	var tmpImage sql.NullString
	var tmpErpID sql.NullInt64
	var tmpErpSchoolID sql.NullString
	var tmpYearGroup sql.NullInt32
	var tmpDivisions sql.NullString

	row := r.Connection.QueryRow("SELECT id, name, surname, is_student, grade, image, isams_id, isams_school_id, year_group, divisions FROM visitors WHERE id = ?", id)
	student := &entity.Visitor{}
	err := row.Scan(
		&student.Id,
		&student.Name,
		&student.Surname,
		&student.IsStudent,
		&tmpGrade,
		&tmpImage,
		&tmpErpID,
		&tmpErpSchoolID,
		&tmpYearGroup,
		&tmpDivisions,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.Visitor{}, nil
		}
		return nil, err
	}
	if tmpGrade.Valid {
		student.Grade = int(tmpGrade.Int32)
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
	return student, nil
}

func (r *Visitor) AddKeyToVisitor(visitor *entity.Visitor, key string) error {
	details, err := r.FindByKey(key)
	if err != nil {
		return fmt.Errorf("Search by key error: %s", err.Error())
	}

	if details.Visitor != nil && details.Visitor.Id > 0 {
		if details.Visitor.Id == visitor.Id {
			return nil
		}
		return fmt.Errorf("Key already assigned to another visitor")
	}

	_, err = r.Connection.Exec("INSERT INTO visitor_key (visitor_id, key_id) VALUES (?, ?)", visitor.Id, key)
	if err != nil {
		return err
	}
	return nil
}

func (r *Visitor) AddVisitor(visitor *entity.Visitor) error {
	if visitor.Id > 0 {
		return r.updateVisitor(visitor)
	} else {
		return r.insertVisitor(visitor)
	}
}

func (r *Visitor) updateVisitor(visitor *entity.Visitor) error {
	var id sql.NullInt32
	if visitor.Id > 0 {
		id = sql.NullInt32{
			Int32: visitor.Id,
			Valid: true,
		}
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
		"UPDATE visitors SET name = ?, surname = ?, is_student = ?, grade = ?, image = ?, isams_id = ?, isams_school_id = ?, year_group = ?, divisions = ?, updated_at = ?, sync_hash = ? WHERE id = ?",
		visitor.Name,
		visitor.Surname,
		visitor.IsStudent,
		visitor.Grade,
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
		"INSERT INTO visitors (name, surname, is_student, grade, image, isams_id, isams_school_id, year_group, divisions, updated_at, sync_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		visitor.Name,
		visitor.Surname,
		visitor.IsStudent,
		visitor.Grade,
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
		return fmt.Errorf("visitor has not be saved correctly: %s", err.Error())
	}
	visitor.Id = int32(id)
	return nil
}

func (r *Visitor) AddRandomImage(student *entity.Visitor) {
	source := rand.NewSource(time.Now().UnixNano())
	rmd := rand.New(source)
	fName := rmd.Intn(11) + 1
	student.Image = fmt.Sprintf("/assets/img/teachers/%d.jpg", fName)
}
