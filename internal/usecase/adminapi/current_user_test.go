package adminapi

import (
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
