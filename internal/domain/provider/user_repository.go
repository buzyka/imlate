package provider

import (
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/google/uuid"
)

type UserRepository interface {
	FindByID(id uuid.UUID) (*entity.User, error)
	FindByUsername(username string) (*entity.User, error)
	FindAll() ([]*entity.User, error)
	Create(user *entity.User) error
	Update(user *entity.User) error
	Delete(id uuid.UUID) error
}
