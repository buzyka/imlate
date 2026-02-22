package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCurrentUserHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
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

	controller := &AdminAPIController{}
	handler := controller.CurrentUserHandler()
	handler(c)

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
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/current-user", nil)

	controller := &AdminAPIController{}
	handler := controller.CurrentUserHandler()
	handler(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCurrentUserHandler_InvalidAuthData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/current-user", nil)

	c.Set("id", "not-a-user-struct")

	controller := &AdminAPIController{}
	handler := controller.CurrentUserHandler()
	handler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
