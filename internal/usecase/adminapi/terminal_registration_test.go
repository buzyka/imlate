package adminapi

import (
	"errors"
	"testing"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newAdminUser(t *testing.T) *entity.User {
	t.Helper()
	u := &entity.User{
		ID:       uuid.New(),
		UserName: "admin",
		Name:     "Admin",
		Surname:  "User",
		Role:     entity.UserRoleAdmin,
		IsActive: true,
	}
	assert.NoError(t, u.SetPassword("admin-password"))
	return u
}

func TestRegisterTerminal_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	mockRepo.On("FindByUsername", "admin").Return(admin, nil)
	mockRepo.On("FindByUsername", "terminal-1").Return(&entity.User{}, nil)
	mockRepo.On("Create", mock.AnythingOfType("*entity.User")).Return(nil)

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.AuthToken)
	assert.Equal(t, "terminal-1", result.TerminalName)
	assert.Equal(t, "terminal-1", result.Username)
	assert.Equal(t, "terminal", result.Role)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_AdminNotFound(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	mockRepo.On("FindByUsername", "unknown").Return(&entity.User{}, nil)

	result, err := api.RegisterTerminal("unknown", "password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAuthenticationFailed)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_AdminRepoError(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	mockRepo.On("FindByUsername", "admin").Return(nil, errors.New("db error"))

	result, err := api.RegisterTerminal("admin", "password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to find admin user")
	assert.NotErrorIs(t, err, ErrAuthenticationFailed)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_WrongPassword(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	mockRepo.On("FindByUsername", "admin").Return(admin, nil)

	result, err := api.RegisterTerminal("admin", "wrong-password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAuthenticationFailed)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_NotAdminRole(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	user := newAdminUser(t)
	user.Role = entity.UserRoleTerminal
	mockRepo.On("FindByUsername", "admin").Return(user, nil)

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAuthenticationFailed)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_InactiveAdmin(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	admin.IsActive = false
	mockRepo.On("FindByUsername", "admin").Return(admin, nil)

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAuthenticationFailed)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_TerminalAlreadyExists_NoForce(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	existingTerminal := &entity.User{
		ID:       uuid.New(),
		UserName: "terminal-1",
		Name:     "terminal-1",
		Role:     entity.UserRoleTerminal,
		IsActive: true,
	}
	mockRepo.On("FindByUsername", "admin").Return(admin, nil)
	mockRepo.On("FindByUsername", "terminal-1").Return(existingTerminal, nil)

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrTerminalAlreadyExists)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_ForceUpdate_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	existingTerminal := &entity.User{
		ID:       uuid.New(),
		UserName: "terminal-1",
		Name:     "terminal-1",
		Role:     entity.UserRoleTerminal,
		IsActive: true,
	}
	assert.NoError(t, existingTerminal.SetPassword("old-password"))
	oldPasswordHash := existingTerminal.Password

	mockRepo.On("FindByUsername", "admin").Return(admin, nil)
	mockRepo.On("FindByUsername", "terminal-1").Return(existingTerminal, nil)
	mockRepo.On("Update", mock.AnythingOfType("*entity.User")).Return(nil)

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", true)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.AuthToken)
	assert.Equal(t, "terminal-1", result.TerminalName)
	assert.Equal(t, "terminal", result.Role)
	assert.NotEqual(t, oldPasswordHash, existingTerminal.Password)
	assert.True(t, existingTerminal.PasswordValidate(result.AuthToken))
	assert.Equal(t, &admin.ID, existingTerminal.CreatedBy)
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_ForceUpdate_NotTerminalRole(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	existingUser := &entity.User{
		ID:       uuid.New(),
		UserName: "terminal-1",
		Name:     "terminal-1",
		Role:     entity.UserRoleAdmin,
		IsActive: true,
	}

	mockRepo.On("FindByUsername", "admin").Return(admin, nil)
	mockRepo.On("FindByUsername", "terminal-1").Return(existingUser, nil)

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", true)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not a terminal")
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_CreateError(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	mockRepo.On("FindByUsername", "admin").Return(admin, nil)
	mockRepo.On("FindByUsername", "terminal-1").Return(&entity.User{}, nil)
	mockRepo.On("Create", mock.AnythingOfType("*entity.User")).Return(errors.New("db error"))

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create terminal")
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_UpdateError(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	existingTerminal := &entity.User{
		ID:       uuid.New(),
		UserName: "terminal-1",
		Name:     "terminal-1",
		Role:     entity.UserRoleTerminal,
		IsActive: true,
	}

	mockRepo.On("FindByUsername", "admin").Return(admin, nil)
	mockRepo.On("FindByUsername", "terminal-1").Return(existingTerminal, nil)
	mockRepo.On("Update", mock.AnythingOfType("*entity.User")).Return(errors.New("db error"))

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", true)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update terminal")
	mockRepo.AssertExpectations(t)
}

func TestRegisterTerminal_CheckExistingTerminalError(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	admin := newAdminUser(t)
	mockRepo.On("FindByUsername", "admin").Return(admin, nil)
	mockRepo.On("FindByUsername", "terminal-1").Return(nil, errors.New("db error"))

	result, err := api.RegisterTerminal("admin", "admin-password", "terminal-1", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check existing terminal")
	mockRepo.AssertExpectations(t)
}

func TestGenerateSecurePassword(t *testing.T) {
	pwd1, err := generateSecurePassword()
	assert.NoError(t, err)
	assert.NotEmpty(t, pwd1)

	pwd2, err := generateSecurePassword()
	assert.NoError(t, err)
	assert.NotEmpty(t, pwd2)

	assert.NotEqual(t, pwd1, pwd2)
	assert.True(t, len(pwd1) >= 32)
}
