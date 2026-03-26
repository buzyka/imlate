package httpauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTerminalUser(t *testing.T, password string) *entity.User {
	t.Helper()
	u := &entity.User{
		ID:       uuid.New(),
		UserName: "term-1",
		Name:     "term-1",
		Role:     entity.UserRoleTerminal,
		IsActive: true,
	}
	assert.NoError(t, u.SetPassword(password))
	return u
}

func setupTerminalAuthTest(repo *providertest.UserRepositoryMock) (*httptest.ResponseRecorder, *gin.Context, *gin.Engine) {
	rec := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(rec)
	m := &TerminalAuthMiddleware{UserRepo: repo}

	engine.POST("/test", m.MiddlewareFunc(), func(c *gin.Context) {
		user, exists := c.Get(TerminalUserContextKey)
		if exists {
			c.JSON(http.StatusOK, gin.H{"user_id": user.(*entity.User).ID.String()})
		} else {
			c.JSON(http.StatusOK, gin.H{"user_id": ""})
		}
	})

	return rec, c, engine
}

func TestTerminalAuth_Success(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	password := "secure-token-123"
	user := newTerminalUser(t, password)
	repo.On("FindByUsername", "term-1").Return(user, nil)

	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	req.Header.Set("Authorization", "Bearer "+password)
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, user.ID.String(), body["user_id"])
	repo.AssertExpectations(t)
}

func TestTerminalAuth_MissingTerminalName(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "Terminal-Name")
}

func TestTerminalAuth_MissingAuthorization(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "Authorization")
}

func TestTerminalAuth_InvalidAuthorizationFormat(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	req.Header.Set("Authorization", "Basic abc123")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "invalid Authorization")
}

func TestTerminalAuth_UserNotFound(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	repo.On("FindByUsername", "unknown").Return(&entity.User{}, nil)

	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "unknown")
	req.Header.Set("Authorization", "Bearer some-token")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "terminal not found")
	repo.AssertExpectations(t)
}

func TestTerminalAuth_RepoError(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	repo.On("FindByUsername", "term-1").Return(nil, errors.New("db error"))

	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	req.Header.Set("Authorization", "Bearer some-token")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	repo.AssertExpectations(t)
}

func TestTerminalAuth_InactiveUser(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	password := "secure-token-123"
	user := newTerminalUser(t, password)
	user.IsActive = false
	repo.On("FindByUsername", "term-1").Return(user, nil)

	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	req.Header.Set("Authorization", "Bearer "+password)
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "not active")
	repo.AssertExpectations(t)
}

func TestTerminalAuth_WrongRole(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	password := "secure-token-123"
	user := newTerminalUser(t, password)
	user.Role = entity.UserRoleAdmin
	repo.On("FindByUsername", "term-1").Return(user, nil)

	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	req.Header.Set("Authorization", "Bearer "+password)
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "not a terminal")
	repo.AssertExpectations(t)
}

func TestTerminalAuth_WrongPassword(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	user := newTerminalUser(t, "correct-password")
	repo.On("FindByUsername", "term-1").Return(user, nil)

	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	req.Header.Set("Authorization", "Bearer wrong-password")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "invalid terminal credentials")
	repo.AssertExpectations(t)
}

func TestTerminalAuth_EmptyBearerToken(t *testing.T) {
	repo := new(providertest.UserRepositoryMock)
	rec, _, engine := setupTerminalAuthTest(repo)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Terminal-Name", "term-1")
	req.Header.Set("Authorization", "Bearer ")
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestParseBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		token  string
		ok     bool
	}{
		{"valid", "Bearer abc123", "abc123", true},
		{"case insensitive", "bearer abc123", "abc123", true},
		{"no space", "Bearerabc123", "", false},
		{"empty token", "Bearer ", "", false},
		{"basic auth", "Basic abc123", "", false},
		{"empty", "", "", false},
		{"only bearer", "Bearer", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := parseBearerToken(tt.header)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.token, token)
		})
	}
}
