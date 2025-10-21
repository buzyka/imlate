package entity

import "time"

type Visitor struct {
	Id      int32  `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Grade   int    `json:"grade"`
	Image   string `json:"image"`
	IsamsId int    `json:"isams_id"`
	IsamsSchoolId int `json:"isams_school_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VisitDetails struct {
	Visitor  *Visitor `json:"visitor"`
	Key 	 string   `json:"key"`
}
