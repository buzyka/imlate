package entity

type VisitorRepository interface {
	FindById(id int32) (*Visitor, error)
	FindByKey(key string) (*VisitDetails, error)
}