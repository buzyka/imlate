package repository

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/isb/entity"
)

type Visitor struct {
	Connection *sql.DB `container:"type"`
}

func (r *Visitor) FindByKey(key string) (*entity.VisitDetails, error) {
	var tmpGrade sql.NullInt32
	var tmpImage sql.NullString
	
	key = strings.ToUpper(key)
	row := r.Connection.QueryRow("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk ON vk.visitor_id = v.id WHERE vk.key_id = ?", key)
	
	visitor := &entity.Visitor{}
	visit := &entity.VisitDetails{
		Visitor: visitor,
	}

	err := row.Scan(
		&visitor.Id, 
		&visitor.Name, 
		&visitor.Surname, 
		&tmpGrade, 
		&tmpImage,
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
	return visit, nil
}

func (r *Visitor) FindById(id int32) (*entity.Visitor, error) {
	var tmpGrade sql.NullInt32
	var tmpImage sql.NullString
	
	row := r.Connection.QueryRow("SELECT id, name, surname, grade, image FROM visitors WHERE id = ?", id)
	student := &entity.Visitor{}
	err := row.Scan(
		&student.Id, 
		&student.Name, 
		&student.Surname, 
		&tmpGrade, 
		&tmpImage,
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
	return student, nil
}

func (r *Visitor) AddKeyToVisitor(visitor *entity.Visitor, key string) error {
	details, err := r.FindByKey(key)
	if err != nil{
		return fmt.Errorf("Search by key error: %s", err.Error())
	}

	if details.Visitor != nil && details.Visitor.Id > 0{
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

func (r *Visitor) AddRandomImage(student *entity.Visitor) {
	source := rand.NewSource(time.Now().UnixNano())
    rmd := rand.New(source)
	fName := rmd.Intn(11)+1
	student.Image = fmt.Sprintf("/assets/img/teachers/%d.jpg", fName)
}
