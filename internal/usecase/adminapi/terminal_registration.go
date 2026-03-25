package adminapi

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/google/uuid"
)

const generatedPasswordLength = 32

var ErrTerminalAlreadyExists = errors.New("terminal with this name already exists")

type TerminalRegistrationResult struct {
	AuthToken    string `json:"auth_token"`
	TerminalName string `json:"terminal_name"`
	Username     string `json:"username"`
	Role         string `json:"role"`
}

func (a *AdminAPI) RegisterTerminal(adminLogin, adminPassword, terminalName string, forceUpdate bool) (*TerminalRegistrationResult, error) {
	admin, err := a.UserRepo.FindByUsername(adminLogin)
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}
	if admin == nil || admin.ID == uuid.Nil {
		return nil, fmt.Errorf("authentication failed")
	}
	if !admin.PasswordValidate(adminPassword) {
		return nil, fmt.Errorf("authentication failed")
	}
	if admin.Role != entity.UserRoleAdmin {
		return nil, fmt.Errorf("authentication failed")
	}
	if !admin.IsActive {
		return nil, fmt.Errorf("authentication failed")
	}

	rawPassword, err := generateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	existing, err := a.UserRepo.FindByUsername(terminalName)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing terminal: %w", err)
	}

	if existing != nil && existing.ID != uuid.Nil {
		if !forceUpdate {
			return nil, ErrTerminalAlreadyExists
		}

		if existing.Role != entity.UserRoleTerminal {
			return nil, fmt.Errorf("user %q exists but is not a terminal", terminalName)
		}

		if err := existing.SetPassword(rawPassword); err != nil {
			return nil, fmt.Errorf("failed to set password: %w", err)
		}
		existing.CreatedBy = &admin.ID
		existing.UpdatedAt = time.Now()

		if err := a.UserRepo.Update(existing); err != nil {
			return nil, fmt.Errorf("failed to update terminal: %w", err)
		}

		return &TerminalRegistrationResult{
			AuthToken:    rawPassword,
			TerminalName: existing.Name,
			Username:     existing.UserName,
			Role:         string(existing.Role),
		}, nil
	}

	terminalUser := &entity.User{
		ID:        uuid.New(),
		UserName:  terminalName,
		Name:      terminalName,
		Surname:   "",
		Role:      entity.UserRoleTerminal,
		IsActive:  true,
		CreatedBy: &admin.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := terminalUser.SetPassword(rawPassword); err != nil {
		return nil, fmt.Errorf("failed to set password: %w", err)
	}

	if err := a.UserRepo.Create(terminalUser); err != nil {
		return nil, fmt.Errorf("failed to create terminal: %w", err)
	}

	return &TerminalRegistrationResult{
		AuthToken:    rawPassword,
		TerminalName: terminalUser.Name,
		Username:     terminalUser.UserName,
		Role:         string(terminalUser.Role),
	}, nil
}

func generateSecurePassword() (string, error) {
	b := make([]byte, generatedPasswordLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
