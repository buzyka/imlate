package adminapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRegHandler(repo *providertest.UserRepositoryMock) (*gin.Engine, *httptest.ResponseRecorder) {
	api := &usecase.AdminAPI{UserRepo: repo}
	controller := &AdminAPIController{AdminAPI: api}

	rec := httptest.NewRecorder()
	engine := gin.New()
	engine.POST("/register-terminal", controller.RegisterTerminalHandler())

	return engine, rec
}

func TestRegisterTerminalHandler_Success(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)

	admin := &entity.User{
		ID:       uuid.New(),
		UserName: "admin",
		Role:     entity.UserRoleAdmin,
		IsActive: true,
	}
	assert.NoError(t, admin.SetPassword("admin-pass"))

	repo.On("FindByUsername", "admin").Return(admin, nil)
	repo.On("FindByUsername", "new-terminal").Return(&entity.User{}, nil)
	repo.On("Create", mock.AnythingOfType("*entity.User")).Return(nil)

	engine, rec := setupRegHandler(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"admin_login":    "admin",
		"admin_password": "admin-pass",
		"terminal_name":  "new-terminal",
		"force_update":   false,
	})
	req := httptest.NewRequest(http.MethodPost, "/register-terminal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp RegisterTerminalResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AuthToken)
	assert.Equal(t, "new-terminal", resp.TerminalName)
	repo.AssertExpectations(t)
}

func TestRegisterTerminalHandler_MissingFields(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	engine, rec := setupRegHandler(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"admin_login": "admin",
	})
	req := httptest.NewRequest(http.MethodPost, "/register-terminal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterTerminalHandler_InvalidJSON(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	engine, rec := setupRegHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/register-terminal", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterTerminalHandler_AuthFailed(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	repo.On("FindByUsername", "admin").Return(&entity.User{}, nil)

	engine, rec := setupRegHandler(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"admin_login":    "admin",
		"admin_password": "wrong",
		"terminal_name":  "term-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/register-terminal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var errResp map[string]string
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "invalid admin credentials", errResp["error"])
	repo.AssertExpectations(t)
}

func TestRegisterTerminalHandler_DBError_Returns500(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	repo.On("FindByUsername", "admin").Return(nil, errors.New("db connection refused"))

	engine, rec := setupRegHandler(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"admin_login":    "admin",
		"admin_password": "admin-pass",
		"terminal_name":  "term-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/register-terminal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp map[string]string
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp["error"], "failed to find admin user")
	repo.AssertExpectations(t)
}

func TestRegisterTerminalHandler_Conflict(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)

	admin := &entity.User{
		ID:       uuid.New(),
		UserName: "admin",
		Role:     entity.UserRoleAdmin,
		IsActive: true,
	}
	assert.NoError(t, admin.SetPassword("admin-pass"))

	existingTerminal := &entity.User{
		ID:       uuid.New(),
		UserName: "term-1",
		Role:     entity.UserRoleTerminal,
		IsActive: true,
	}

	repo.On("FindByUsername", "admin").Return(admin, nil)
	repo.On("FindByUsername", "term-1").Return(existingTerminal, nil)

	engine, rec := setupRegHandler(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"admin_login":    "admin",
		"admin_password": "admin-pass",
		"terminal_name":  "term-1",
		"force_update":   false,
	})
	req := httptest.NewRequest(http.MethodPost, "/register-terminal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	repo.AssertExpectations(t)
}

func TestRegisterTerminalHandler_ForceUpdateSuccess(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)

	admin := &entity.User{
		ID:       uuid.New(),
		UserName: "admin",
		Role:     entity.UserRoleAdmin,
		IsActive: true,
	}
	assert.NoError(t, admin.SetPassword("admin-pass"))

	existingTerminal := &entity.User{
		ID:       uuid.New(),
		UserName: "term-1",
		Name:     "term-1",
		Role:     entity.UserRoleTerminal,
		IsActive: true,
	}

	repo.On("FindByUsername", "admin").Return(admin, nil)
	repo.On("FindByUsername", "term-1").Return(existingTerminal, nil)
	repo.On("Update", mock.AnythingOfType("*entity.User")).Return(nil)

	engine, rec := setupRegHandler(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"admin_login":    "admin",
		"admin_password": "admin-pass",
		"terminal_name":  "term-1",
		"force_update":   true,
	})
	req := httptest.NewRequest(http.MethodPost, "/register-terminal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp RegisterTerminalResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AuthToken)
	assert.Equal(t, "term-1", resp.TerminalName)
	repo.AssertExpectations(t)
}
