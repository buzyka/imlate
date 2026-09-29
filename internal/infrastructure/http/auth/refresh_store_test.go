package httpauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/appleboy/gin-jwt/v3/core"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testRefreshToken = "raw-refresh-token"

func newRefreshStoreForTests() (*DBRefreshTokenStore, *providertest.RefreshTokenRepositoryMock, *providertest.UserRepositoryMock) {
	repo := new(providertest.RefreshTokenRepositoryMock)
	userRepo := new(providertest.UserRepositoryMock)
	return &DBRefreshTokenStore{Repo: repo, UserRepo: userRepo}, repo, userRepo
}

func TestHashRefreshToken(t *testing.T) {
	hash := hashRefreshToken(testRefreshToken)

	assert.Len(t, hash, 64)
	assert.NotContains(t, hash, testRefreshToken)
	assert.Equal(t, hash, hashRefreshToken(testRefreshToken))
	assert.NotEqual(t, hash, hashRefreshToken("other-token"))
}

func TestDBRefreshTokenStore_Set_Success(t *testing.T) {
	store, repo, _ := newRefreshStoreForTests()
	userID := uuid.New()
	expiry := time.Now().Add(time.Hour)
	repo.On("Save", hashRefreshToken(testRefreshToken), userID, expiry).Return(nil)

	err := store.Set(context.Background(), testRefreshToken, &entity.User{ID: userID}, expiry)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDBRefreshTokenStore_Set_RepoError(t *testing.T) {
	store, repo, _ := newRefreshStoreForTests()
	repo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := store.Set(context.Background(), testRefreshToken, &entity.User{ID: uuid.New()}, time.Now())

	assert.EqualError(t, err, "db error")
	repo.AssertExpectations(t)
}

func TestDBRefreshTokenStore_Set_UnsupportedUserData(t *testing.T) {
	var nilUser *entity.User
	tests := map[string]any{
		"not a user":   "user-id",
		"nil user":     nilUser,
		"user no id":   &entity.User{},
		"nil userData": nil,
	}
	for name, userData := range tests {
		t.Run(name, func(t *testing.T) {
			store, repo, _ := newRefreshStoreForTests()

			err := store.Set(context.Background(), testRefreshToken, userData, time.Now())

			assert.ErrorContains(t, err, "unsupported refresh token user data")
			repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestDBRefreshTokenStore_Get_ReturnsFreshUser(t *testing.T) {
	store, repo, userRepo := newRefreshStoreForTests()
	user := &entity.User{ID: uuid.New(), UserName: "admin", IsActive: true}
	repo.On("FindUserID", hashRefreshToken(testRefreshToken), mock.Anything).Return(user.ID, nil)
	userRepo.On("FindByID", user.ID).Return(user, nil)

	data, err := store.Get(context.Background(), testRefreshToken)

	assert.NoError(t, err)
	assert.Same(t, user, data)
	repo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestDBRefreshTokenStore_Get_NotFound(t *testing.T) {
	store, repo, userRepo := newRefreshStoreForTests()
	repo.On("FindUserID", mock.Anything, mock.Anything).Return(uuid.Nil, provider.ErrRefreshTokenNotFound)

	data, err := store.Get(context.Background(), testRefreshToken)

	assert.Nil(t, data)
	assert.ErrorIs(t, err, core.ErrRefreshTokenNotFound)
	userRepo.AssertNotCalled(t, "FindByID", mock.Anything)
}

func TestDBRefreshTokenStore_Get_RepoError(t *testing.T) {
	store, repo, _ := newRefreshStoreForTests()
	repo.On("FindUserID", mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("db error"))

	data, err := store.Get(context.Background(), testRefreshToken)

	assert.Nil(t, data)
	assert.EqualError(t, err, "db error")
}

func TestDBRefreshTokenStore_Get_UserRepoError(t *testing.T) {
	store, repo, userRepo := newRefreshStoreForTests()
	userID := uuid.New()
	repo.On("FindUserID", mock.Anything, mock.Anything).Return(userID, nil)
	userRepo.On("FindByID", userID).Return(nil, errors.New("db error"))

	data, err := store.Get(context.Background(), testRefreshToken)

	assert.Nil(t, data)
	assert.EqualError(t, err, "failed to load refresh token user: db error")
}

func TestDBRefreshTokenStore_Get_RevokesTokenOfUnavailableUser(t *testing.T) {
	tests := map[string]*entity.User{
		"nil user":      nil,
		"deleted user":  {}, // UserRepo returns an empty user when no row matches
		"inactive user": {ID: uuid.New(), IsActive: false},
	}
	for name, user := range tests {
		t.Run(name, func(t *testing.T) {
			store, repo, userRepo := newRefreshStoreForTests()
			userID := uuid.New()
			tokenHash := hashRefreshToken(testRefreshToken)
			repo.On("FindUserID", tokenHash, mock.Anything).Return(userID, nil)
			if user == nil {
				userRepo.On("FindByID", userID).Return(nil, nil)
			} else {
				userRepo.On("FindByID", userID).Return(user, nil)
			}
			repo.On("Delete", tokenHash).Return(nil)

			data, err := store.Get(context.Background(), testRefreshToken)

			assert.Nil(t, data)
			assert.ErrorIs(t, err, core.ErrRefreshTokenNotFound)
			repo.AssertExpectations(t)
		})
	}
}

func TestDBRefreshTokenStore_Get_RevokeError(t *testing.T) {
	store, repo, userRepo := newRefreshStoreForTests()
	userID := uuid.New()
	repo.On("FindUserID", mock.Anything, mock.Anything).Return(userID, nil)
	userRepo.On("FindByID", userID).Return(&entity.User{ID: userID, IsActive: false}, nil)
	repo.On("Delete", mock.Anything).Return(errors.New("db error"))

	data, err := store.Get(context.Background(), testRefreshToken)

	assert.Nil(t, data)
	assert.EqualError(t, err, "db error")
}

func TestDBRefreshTokenStore_Delete(t *testing.T) {
	store, repo, _ := newRefreshStoreForTests()
	repo.On("Delete", hashRefreshToken(testRefreshToken)).Return(nil)

	assert.NoError(t, store.Delete(context.Background(), testRefreshToken))
	repo.AssertExpectations(t)
}

func TestDBRefreshTokenStore_Cleanup(t *testing.T) {
	store, repo, _ := newRefreshStoreForTests()
	repo.On("DeleteExpired", mock.Anything).Return(3, nil)

	deleted, err := store.Cleanup(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 3, deleted)
	repo.AssertExpectations(t)
}

func TestDBRefreshTokenStore_Count(t *testing.T) {
	store, repo, _ := newRefreshStoreForTests()
	repo.On("CountActive", mock.Anything).Return(5, nil)

	count, err := store.Count(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	repo.AssertExpectations(t)
}

// fakeRefreshTokenRepo is an in-memory RefreshTokenRepository for the end-to-end flow test.
type fakeRefreshTokenRepo struct {
	mu     sync.Mutex
	tokens map[string]uuid.UUID
}

func (f *fakeRefreshTokenRepo) Save(tokenHash string, userID uuid.UUID, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens[tokenHash] = userID
	return nil
}

func (f *fakeRefreshTokenRepo) FindUserID(tokenHash string, _ time.Time) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, ok := f.tokens[tokenHash]
	if !ok {
		return uuid.Nil, provider.ErrRefreshTokenNotFound
	}
	return userID, nil
}

func (f *fakeRefreshTokenRepo) Delete(tokenHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tokens, tokenHash)
	return nil
}

func (f *fakeRefreshTokenRepo) DeleteByUserID(uuid.UUID) error       { return nil }
func (f *fakeRefreshTokenRepo) DeleteExpired(time.Time) (int, error) { return 0, nil }
func (f *fakeRefreshTokenRepo) CountActive(time.Time) (int, error)   { return 0, nil }

func TestAuthFlow_LoginRefreshLogoutWithDBStore(t *testing.T) {
	gintestSetup()
	userRepo := new(providertest.UserRepositoryMock)
	refreshRepo := &fakeRefreshTokenRepo{tokens: map[string]uuid.UUID{}}
	manager := newAuthManagerForTests(userRepo, "0123456789abcdef0123456789abcdef")
	manager.RefreshRepo = refreshRepo

	admin := &entity.User{ID: uuid.New(), UserName: "admin", Role: entity.UserRoleAdmin, IsActive: true}
	assert.NoError(t, admin.SetPassword("password123"))
	userRepo.On("FindByUsername", "admin").Return(admin, nil)
	userRepo.On("FindByID", admin.ID).Return(admin, nil)

	middleware, err := manager.Init()
	assert.NoError(t, err)
	r := gin.New()
	r.POST("/login", middleware.LoginHandler)
	r.POST("/refresh", middleware.RefreshHandler)
	r.POST("/logout", middleware.LogoutHandler)

	post := func(path, body string) (int, map[string]any) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(rec, req)
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return rec.Code, resp
	}

	code, login := post("/login", `{"username":"admin","password":"password123"}`)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(30*60), login["expires_in"])
	firstRefresh := login["refresh_token"].(string)
	assert.Len(t, refreshRepo.tokens, 1)
	assert.Contains(t, refreshRepo.tokens, hashRefreshToken(firstRefresh))

	code, refreshed := post("/refresh", `{"refresh_token":"`+firstRefresh+`"}`)
	assert.Equal(t, http.StatusOK, code)
	secondRefresh := refreshed["refresh_token"].(string)
	assert.NotEqual(t, firstRefresh, secondRefresh)

	// Rotation: the old refresh token is revoked.
	code, _ = post("/refresh", `{"refresh_token":"`+firstRefresh+`"}`)
	assert.Equal(t, http.StatusUnauthorized, code)

	code, _ = post("/logout", `{"refresh_token":"`+secondRefresh+`"}`)
	assert.Equal(t, http.StatusOK, code)
	assert.Empty(t, refreshRepo.tokens)

	code, _ = post("/refresh", `{"refresh_token":"`+secondRefresh+`"}`)
	assert.Equal(t, http.StatusUnauthorized, code)
}
