package adminapi

import (
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetCurrentUser_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	expectedUser := &entity.User{
		ID:        userID,
		UserName:  "admin",
		Name:      "John",
		Surname:   "Doe",
		Role:      entity.UserRoleAdmin,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("FindByID", userID).Return(expectedUser, nil)

	user, err := api.GetCurrentUser(userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestGetCurrentUser_NotFound(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	mockRepo.On("FindByID", userID).Return(nil, errors.New("user not found"))

	user, err := api.GetCurrentUser(userID)

	assert.Error(t, err)
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestGetAllUsers_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	users := []*entity.User{
		{ID: uuid.New(), UserName: "user1"},
		{ID: uuid.New(), UserName: "user2"},
	}
	mockRepo.On("FindAll").Return(users, nil)

	result, err := api.GetAllUsers()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetAllUsers_Error(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	mockRepo.On("FindAll").Return(nil, errors.New("db error"))

	result, err := api.GetAllUsers()

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCreateUser_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	mockRepo.On("FindByUsername", "newuser").Return(&entity.User{}, nil)
	mockRepo.On("Create", mock.AnythingOfType("*entity.User")).Return(nil)

	user, err := api.CreateUser("newuser", "password123", "Jane", "Doe", entity.UserRoleAdmin)

	assert.NoError(t, err)
	assert.Equal(t, "newuser", user.UserName)
	assert.Equal(t, "Jane", user.Name)
	assert.Equal(t, "Doe", user.Surname)
	assert.Equal(t, entity.UserRoleAdmin, user.Role)
	assert.True(t, user.IsActive)
	assert.True(t, user.PasswordValidate("password123"))
	mockRepo.AssertExpectations(t)
}

func TestCreateUser_FindByUsernameError(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	mockRepo.On("FindByUsername", "newuser").Return(nil, errors.New("db down"))

	user, err := api.CreateUser("newuser", "password123", "Jane", "Doe", entity.UserRoleAdmin)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to check existing username")
	mockRepo.AssertExpectations(t)
}

func TestCreateUser_SuccessWhenEmptyUserReturned(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	// Simulates scanUser returning &entity.User{} with zero UUID when no rows found
	mockRepo.On("FindByUsername", "newuser").Return(&entity.User{}, nil)
	mockRepo.On("Create", mock.AnythingOfType("*entity.User")).Return(nil)

	user, err := api.CreateUser("newuser", "password123", "Jane", "Doe", entity.UserRoleAdmin)

	assert.NoError(t, err)
	assert.Equal(t, "newuser", user.UserName)
	mockRepo.AssertExpectations(t)
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	existing := &entity.User{ID: uuid.New(), UserName: "existing"}
	mockRepo.On("FindByUsername", "existing").Return(existing, nil)

	user, err := api.CreateUser("existing", "pass", "A", "B", entity.UserRoleAdmin)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "already exists")
	mockRepo.AssertExpectations(t)
}

func TestUpdateUser_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	existing := &entity.User{ID: userID, UserName: "admin", Name: "Old", Surname: "Name"}
	mockRepo.On("FindByID", userID).Return(existing, nil)
	mockRepo.On("Update", mock.AnythingOfType("*entity.User")).Return(nil)

	user, err := api.UpdateUser(userID, "New", "Name", entity.UserRoleTerminal, false)

	assert.NoError(t, err)
	assert.Equal(t, "New", user.Name)
	assert.Equal(t, "Name", user.Surname)
	assert.Equal(t, entity.UserRoleTerminal, user.Role)
	assert.False(t, user.IsActive)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUser_NotFound(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	mockRepo.On("FindByID", userID).Return(nil, errors.New("not found"))

	user, err := api.UpdateUser(userID, "A", "B", entity.UserRoleAdmin, true)

	assert.Error(t, err)
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	mockRepo.On("Delete", userID).Return(nil)

	err := api.DeleteUser(userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteUser_Error(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	mockRepo.On("Delete", userID).Return(errors.New("db error"))

	err := api.DeleteUser(userID)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdatePassword_Success(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	existing := &entity.User{ID: userID, UserName: "admin"}
	_ = existing.SetPassword("oldpassword")
	mockRepo.On("FindByID", userID).Return(existing, nil)
	mockRepo.On("Update", mock.AnythingOfType("*entity.User")).Return(nil)

	err := api.UpdatePassword(userID, "newpassword123")

	assert.NoError(t, err)
	assert.True(t, existing.PasswordValidate("newpassword123"))
	assert.False(t, existing.PasswordValidate("oldpassword"))
	mockRepo.AssertExpectations(t)
}

func TestUpdatePassword_UserNotFound(t *testing.T) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &AdminAPI{UserRepo: mockRepo}

	userID := uuid.New()
	mockRepo.On("FindByID", userID).Return(nil, errors.New("not found"))

	err := api.UpdatePassword(userID, "newpassword")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
	mockRepo.AssertExpectations(t)
}
