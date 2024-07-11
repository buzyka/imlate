package entity

type VisitorRepository interface {
	FindById(id string) (*Visitor, error)
}