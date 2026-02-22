package adminapi

import (
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/google/uuid"
)

type AdminAPI struct {
	UserRepo provider.UserRepository `container:"type"`
}

func (a *AdminAPI) GetCurrentUser(userID uuid.UUID) (*entity.User, error) {
	return a.UserRepo.FindByID(userID)
}
