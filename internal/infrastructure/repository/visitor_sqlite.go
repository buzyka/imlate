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

func (r *Visitor) FindById(id string) (*entity.Visitor, error) {
	var tmpGrade sql.NullInt32
	var tmpImage sql.NullString
	
	id = strings.ToUpper(id)
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

func (r *Visitor) AddRandomImage(student *entity.Visitor) {
	source := rand.NewSource(time.Now().UnixNano())
    rmd := rand.New(source)
	fName := rmd.Intn(11)+1
	student.Image = fmt.Sprintf("/assets/img/teachers/%d.jpg", fName)
}
