package adminapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTest() (*providertest.UserRepositoryMock, *AdminAPIController) {
	mockRepo := new(providertest.UserRepositoryMock)
	api := &usecase.AdminAPI{UserRepo: mockRepo}
	controller := &AdminAPIController{AdminAPI: api}
	gin.SetMode(gin.TestMode)
	return mockRepo, controller
}

func TestCurrentUserHandler_Success(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	expectedUser := &entity.User{
		ID:        userID,
		UserName:  "admin",
		Name:      "John",
		Surname:   "Doe",
		Role:      entity.UserRoleAdmin,
		IsActive:  true,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	c.Set("id", expectedUser)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/current-user", nil)

	controller.CurrentUserHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseUser entity.User
	err := json.Unmarshal(w.Body.Bytes(), &responseUser)
	assert.NoError(t, err)
	assert.Equal(t, userID, responseUser.ID)
	assert.Equal(t, "admin", responseUser.UserName)
	assert.Equal(t, "John", responseUser.Name)
	assert.Equal(t, "Doe", responseUser.Surname)
	assert.Equal(t, entity.UserRoleAdmin, responseUser.Role)
	assert.Empty(t, responseUser.Password)
}

func TestCurrentUserHandler_NoAuthContext(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/current-user", nil)

	controller.CurrentUserHandler()(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCurrentUserHandler_InvalidAuthData(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/current-user", nil)
	c.Set("id", "not-a-user-struct")

	controller.CurrentUserHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListUsersHandler_Success(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/users", nil)

	users := []*entity.User{
		{ID: uuid.New(), UserName: "user1"},
		{ID: uuid.New(), UserName: "user2"},
	}
	mockRepo.On("FindAll").Return(users, nil)

	controller.ListUsersHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []entity.User
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListUsersHandler_Error(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/users", nil)

	mockRepo.On("FindAll").Return(nil, errors.New("db error"))

	controller.ListUsersHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetUserHandler_Success(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/users/"+userID.String(), nil)

	user := &entity.User{ID: userID, UserName: "admin"}
	mockRepo.On("FindByID", userID).Return(user, nil)

	controller.GetUserHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserHandler_InvalidID(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/users/bad", nil)

	controller.GetUserHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetUserHandler_NotFound(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/users/"+userID.String(), nil)

	mockRepo.On("FindByID", userID).Return(nil, errors.New("not found"))

	controller.GetUserHandler()(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateUserHandler_Success(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(CreateUserRequest{
		Username: "newuser",
		Password: "password123",
		Name:     "Jane",
		Surname:  "Doe",
		Role:     entity.UserRoleAdmin,
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/users", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	mockRepo.On("FindByUsername", "newuser").Return(nil, errors.New("not found"))
	mockRepo.On("Create", mock.AnythingOfType("*entity.User")).Return(nil)

	controller.CreateUserHandler()(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateUserHandler_BadRequest(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/users", bytes.NewReader([]byte("{}")))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.CreateUserHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserHandler_Success(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	body, _ := json.Marshal(UpdateUserRequest{
		Name:     "Updated",
		Surname:  "Name",
		Role:     entity.UserRoleTerminal,
		IsActive: true,
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/users/"+userID.String(), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	existing := &entity.User{ID: userID, UserName: "admin", Name: "Old", Surname: "Name"}
	mockRepo.On("FindByID", userID).Return(existing, nil)
	mockRepo.On("Update", mock.AnythingOfType("*entity.User")).Return(nil)

	controller.UpdateUserHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateUserHandler_InvalidID(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/users/bad-id", nil)

	controller.UpdateUserHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteUserHandler_Success(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/users/"+userID.String(), nil)

	mockRepo.On("Delete", userID).Return(nil)

	controller.DeleteUserHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteUserHandler_InvalidID(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/users/bad-id", nil)

	controller.DeleteUserHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteUserHandler_Error(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/users/"+userID.String(), nil)

	mockRepo.On("Delete", userID).Return(errors.New("db error"))

	controller.DeleteUserHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdatePasswordHandler_Success(t *testing.T) {
	mockRepo, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	body, _ := json.Marshal(UpdatePasswordRequest{Password: "newpassword123"})
	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/users/"+userID.String()+"/password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	existing := &entity.User{ID: userID, UserName: "admin"}
	mockRepo.On("FindByID", userID).Return(existing, nil)
	mockRepo.On("Update", mock.AnythingOfType("*entity.User")).Return(nil)

	controller.UpdatePasswordHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdatePasswordHandler_InvalidID(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/users/bad-id/password", nil)

	controller.UpdatePasswordHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdatePasswordHandler_TooShort(t *testing.T) {
	_, controller := setupTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID := uuid.New()
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	body, _ := json.Marshal(UpdatePasswordRequest{Password: "short"})
	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/users/"+userID.String()+"/password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.UpdatePasswordHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
