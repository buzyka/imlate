package adminapi

import (
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/google/uuid"
)

type AdminAPI struct {
	UserRepo             provider.UserRepository             `container:"type"`
	VisitorRepo          provider.VisitorRepository          `container:"type"`
	VisitorTrackRepo     provider.VisitorTrackRepository     `container:"type"`
	VisitDailyReportRepo provider.VisitDailyReportRepository `container:"type"`
	RefreshTokenRepo     provider.RefreshTokenRepository     `container:"type"`
	Config               *config.Config                      `container:"type"`
}

func (a *AdminAPI) GetCurrentUser(userID uuid.UUID) (*entity.User, error) {
	return a.UserRepo.FindByID(userID)
}

func (a *AdminAPI) GetAllUsers() ([]*entity.User, error) {
	return a.UserRepo.FindAll()
}

func (a *AdminAPI) GetUser(id uuid.UUID) (*entity.User, error) {
	return a.UserRepo.FindByID(id)
}

func (a *AdminAPI) CreateUser(username, password, name, surname string, role entity.UserRole) (*entity.User, error) {
	existing, err := a.UserRepo.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing username: %w", err)
	}

	if existing != nil && existing.ID != uuid.Nil {
		return nil, fmt.Errorf("user with username %q already exists", username)
	}

	user := &entity.User{
		ID:        uuid.New(),
		UserName:  username,
		Name:      name,
		Surname:   surname,
		Role:      role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.SetPassword(password); err != nil {
		return nil, fmt.Errorf("failed to set password: %w", err)
	}

	if err := a.UserRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (a *AdminAPI) UpdateUser(id uuid.UUID, name, surname string, role entity.UserRole, isActive bool) (*entity.User, error) {
	user, err := a.UserRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// A session must not outlive the access it was issued for.
	accessChanged := user.Role != role || (user.IsActive && !isActive)

	user.Name = name
	user.Surname = surname
	user.Role = role
	user.IsActive = isActive
	user.UpdatedAt = time.Now()

	if err := a.UserRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if accessChanged {
		if err := a.revokeSessions(user.ID); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (a *AdminAPI) UpdatePassword(id uuid.UUID, password string) error {
	user, err := a.UserRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := user.SetPassword(password); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	user.UpdatedAt = time.Now()

	if err := a.UserRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return a.revokeSessions(user.ID)
}

func (a *AdminAPI) DeleteUser(id uuid.UUID) error {
	if err := a.UserRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	// Users are soft-deleted, so the FK cascade on refresh tokens does not fire.
	return a.revokeSessions(id)
}

// revokeSessions invalidates all refresh tokens of the user, forcing a new login.
func (a *AdminAPI) revokeSessions(userID uuid.UUID) error {
	if err := a.RefreshTokenRepo.DeleteByUserID(userID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}
	return nil
}
