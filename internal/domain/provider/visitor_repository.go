package provider

import "github.com/buzyka/imlate/internal/domain/entity"

type VisitorFilterConfig struct {
	ERPYearGroup []*int32
}

type VisitorFilterOption func(*VisitorFilterConfig)

func WithERPYearGroups(yearGroups []int32) VisitorFilterOption {
	return func(cfg *VisitorFilterConfig) {
		cfg.ERPYearGroup = make([]*int32, 0, len(yearGroups))
		for _, yg := range yearGroups {
			ygCopy := yg
			cfg.ERPYearGroup = append(cfg.ERPYearGroup, &ygCopy)
		}
	}
}

type VisitorRepository interface {
	GetAll() ([]*entity.Visitor, error)
	FindAll(opts ...VisitorFilterOption) ([]*entity.Visitor, error)
	FindById(id int32) (*entity.Visitor, error)
	FindByKey(key string) (*entity.VisitDetails, error)

	AddKeyToVisitor(visitor *entity.Visitor, key string) error
	RemoveKeyFromVisitor(visitorID int32, key string) error
	FindKeysByVisitorId(visitorID int32) ([]string, error)
	AddVisitor(visitor *entity.Visitor) error
	SaveVisitor(visitor *entity.Visitor) error
	DeleteVisitor(id int32) error
	UpdateVisitorImage(id int32, imagePath string) error
}
