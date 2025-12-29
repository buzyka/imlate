package entity

import "time"

type VisitTrack struct {
	Id int `json:"id"`
	VisitorId int32 `json:"visitor_id"`
	VisitKey  string `json:"visit_key"`
	Visitor *Visitor
	CreatedAt time.Time `json:"created_at"`
	SignedIn bool `json:"signed_in"`
}  