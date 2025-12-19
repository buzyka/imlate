package entity

import "time"

type Visitor struct {
	Id      int32  `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	IsStudent bool `json:"is_student"`
	Grade   int    `json:"grade"`	
	Image   string `json:"image"`
	ErpID   int64 `json:"isams_id"`
	ErpSchoolID string `json:"isams_school_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VisitDetails struct {
	Visitor  *Visitor `json:"visitor"`
	Key 	 string   `json:"key"`
}
