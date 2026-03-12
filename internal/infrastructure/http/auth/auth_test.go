package httpauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gjwt "github.com/appleboy/gin-jwt/v3"
	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const testJWTPayloadKey = "JWT_PAYLOAD"

func newAuthManagerForTests(repo *providertest.UserRepositoryMock, secret string) *AuthManager {
	return &AuthManager{
		Config:   &config.Config{AuthTokenSecret: secret},
		UserRepo: repo,
	}
}

func TestInit_AuthTokenSecretTooShort(t *testing.T) {
	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "short")

	middleware, err := manager.Init()

	assert.Nil(t, middleware)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "AUTH_TOKEN_SECRET must be at least")
}

func TestInit_ConfigNil(t *testing.T) {
	manager := &AuthManager{UserRepo: new(providertest.UserRepositoryMock)}

	middleware, err := manager.Init()

	assert.Nil(t, middleware)
	assert.EqualError(t, err, "auth config is not set")
}

func TestInit_AuthTokenSecretValid(t *testing.T) {
	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")

	middleware, err := manager.Init()

	assert.NoError(t, err)
	assert.NotNil(t, middleware)
}

func TestInit_NewMiddlewareError(t *testing.T) {
	origNew := newJWTMiddleware
	t.Cleanup(func() {
		newJWTMiddleware = origNew
	})

	newJWTMiddleware = func(*gjwt.GinJWTMiddleware) (*gjwt.GinJWTMiddleware, error) {
		return nil, errors.New("boom")
	}

	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")

	middleware, err := manager.Init()

	assert.Nil(t, middleware)
	assert.EqualError(t, err, "JWT Error: boom")
}

func TestInit_MiddlewareInitError(t *testing.T) {
	origInit := initJWTMiddleware
	t.Cleanup(func() {
		initJWTMiddleware = origInit
	})

	initJWTMiddleware = func(*gjwt.GinJWTMiddleware) error {
		return errors.New("init failed")
	}

	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")

	middleware, err := manager.Init()

	assert.Nil(t, middleware)
	assert.EqualError(t, err, "authMiddleware.MiddlewareInit() Error: init failed")
}

func TestPayloadFunc_UserAndNonUser(t *testing.T) {
	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")
	userID := uuid.New()

	claimsUser := manager.payloadFunc()(&entity.User{ID: userID})
	claimsOther := manager.payloadFunc()("not-user")

	assert.Equal(t, userID.String(), claimsUser[identityKey])
	assert.Empty(t, claimsOther)
}

func TestAuthenticator_InactiveUserReturnsForbidden(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	inactiveUser := &entity.User{UserName: "inactive", IsActive: false}
	err := inactiveUser.SetPassword("password123")
	assert.NoError(t, err)

	repo.On("FindByUsername", "inactive").Return(inactiveUser, nil)

	body := []byte(`{"username":"inactive","password":"password123"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, authErr := manager.authenticator()(c)

	assert.Nil(t, result)
	assert.ErrorIs(t, authErr, gjwt.ErrForbidden)
	repo.AssertExpectations(t)
}

func TestIdentityHandler_InvalidClaimReturnsNil(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{identityKey: 42})

	assert.NotPanics(t, func() {
		identity := manager.identityHandler()(c)
		assert.Nil(t, identity)
	})
}

func TestIdentityHandler_MissingClaimReturnsNil(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{})

	identity := manager.identityHandler()(c)

	assert.Nil(t, identity)
}

func TestIdentityHandler_EmptyClaimReturnsNil(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{identityKey: ""})

	identity := manager.identityHandler()(c)

	assert.Nil(t, identity)
}

func TestIdentityHandler_InvalidUUIDReturnsNil(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{identityKey: "not-a-uuid"})

	identity := manager.identityHandler()(c)

	assert.Nil(t, identity)
}

func TestIdentityHandler_RepoErrorReturnsNil(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	userID := uuid.New()
	repo.On("FindByID", userID).Return(nil, errors.New("db error"))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{identityKey: userID.String()})

	identity := manager.identityHandler()(c)

	assert.Nil(t, identity)
	repo.AssertExpectations(t)
}

func TestIdentityHandler_RepoNilUserReturnsNil(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	userID := uuid.New()
	repo.On("FindByID", userID).Return(nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{identityKey: userID.String()})

	identity := manager.identityHandler()(c)

	assert.Nil(t, identity)
	repo.AssertExpectations(t)
}

func TestIdentityHandler_ValidClaimReturnsUser(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	userID := uuid.New()
	expectedUser := &entity.User{ID: userID, UserName: "admin", IsActive: true}
	repo.On("FindByID", userID).Return(expectedUser, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{identityKey: userID.String()})

	identity := manager.identityHandler()(c)

	assert.Same(t, expectedUser, identity)
	repo.AssertExpectations(t)
}

func TestAuthenticator_MissingLoginValues(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	result, authErr := manager.authenticator()(c)

	assert.Equal(t, "", result)
	assert.ErrorIs(t, authErr, gjwt.ErrMissingLoginValues)
}

func TestAuthenticator_FindByUsernameError(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")
	repo.On("FindByUsername", "admin").Return(nil, errors.New("db down"))

	body := []byte(`{"username":"admin","password":"password123"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, authErr := manager.authenticator()(c)

	assert.Nil(t, result)
	assert.ErrorIs(t, authErr, gjwt.ErrFailedAuthentication)
	repo.AssertExpectations(t)
}

func TestAuthenticator_WrongPassword(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	user := &entity.User{UserName: "admin", IsActive: true}
	err := user.SetPassword("correct-password")
	assert.NoError(t, err)

	repo.On("FindByUsername", "admin").Return(user, nil)

	body := []byte(`{"username":"admin","password":"wrong-password"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, authErr := manager.authenticator()(c)

	assert.Nil(t, result)
	assert.ErrorIs(t, authErr, gjwt.ErrFailedAuthentication)
	repo.AssertExpectations(t)
}

func TestAuthenticator_Success(t *testing.T) {
	gintestSetup()
	repo := new(providertest.UserRepositoryMock)
	manager := newAuthManagerForTests(repo, "0123456789abcdef0123456789abcdef")

	user := &entity.User{UserName: "admin", IsActive: true}
	err := user.SetPassword("password123")
	assert.NoError(t, err)

	repo.On("FindByUsername", "admin").Return(user, nil)

	body := []byte(`{"username":"admin","password":"password123"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, authErr := manager.authenticator()(c)

	assert.NoError(t, authErr)
	assert.Same(t, user, result)
	repo.AssertExpectations(t)
}

func TestAuthorizer(t *testing.T) {
	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")

	assert.True(t, manager.authorizer()(nil, &entity.User{Role: entity.UserRoleAdmin}))
	assert.False(t, manager.authorizer()(nil, &entity.User{Role: entity.UserRoleTerminal}))
	assert.False(t, manager.authorizer()(nil, "not-a-user"))
}

func TestUnauthorizedResponse(t *testing.T) {
	gintestSetup()
	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	manager.unauthorized()(c, http.StatusUnauthorized, "unauthorized")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, float64(http.StatusUnauthorized), body["code"])
	assert.Equal(t, "unauthorized", body["message"])
}

func TestLogoutResponse_WithClaimsAndUser(t *testing.T) {
	gintestSetup()
	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(testJWTPayloadKey, jwt.MapClaims{identityKey: "u-1"})
	c.Set(identityKey, &entity.User{UserName: "admin"})

	manager.logoutResponse()(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, float64(http.StatusOK), body["code"])
	assert.Equal(t, "Successfully logged out", body["message"])
	assert.Equal(t, "u-1", body["logged_out_user"])
	assert.Equal(t, "admin", body["user_info"])
}

func TestLogoutResponse_WithoutClaimsAndUser(t *testing.T) {
	gintestSetup()
	manager := newAuthManagerForTests(new(providertest.UserRepositoryMock), "0123456789abcdef0123456789abcdef")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	manager.logoutResponse()(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, float64(http.StatusOK), body["code"])
	assert.Equal(t, "Successfully logged out", body["message"])
	_, hasLoggedOutUser := body["logged_out_user"]
	_, hasUserInfo := body["user_info"]
	assert.False(t, hasLoggedOutUser)
	assert.False(t, hasUserInfo)
}

func gintestSetup() {
	gin.SetMode(gin.TestMode)
}
