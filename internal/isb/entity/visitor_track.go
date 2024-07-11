package entity

import "time"

type VisitTrack struct {
	Id int `json:"id"`
	VisitorId string `json:"visitor_id"`
	Visitor *Visitor
	CreatedAt time.Time `json:"created_at"`
	SignedIn bool `json:"signed_in"`
}  