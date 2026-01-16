package provider

import "github.com/buzyka/imlate/internal/domain/entity"


type VisitorRepository interface {	
	GetAll() ([]*entity.Visitor, error)
	FindById(id int32) (*entity.Visitor, error)
	FindByKey(key string) (*entity.VisitDetails, error)
	
	AddKeyToVisitor(visitor *entity.Visitor, key string) error
	AddVisitor(visitor *entity.Visitor) error
}
