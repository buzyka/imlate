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

func setupTrackTest() (*providertest.VisitorRepositoryMock, *providertest.VisitorTrackRepositoryMock, *AdminAPIController) {
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	api := &usecase.AdminAPI{VisitorRepo: mockVisitorRepo}
	controller := &AdminAPIController{
		AdminAPI:  api,
		TrackRepo: mockTrackRepo,
	}
	gin.SetMode(gin.TestMode)
	return mockVisitorRepo, mockTrackRepo, controller
}

func TestManualTrackHandler_Success_SignIn(t *testing.T) {
	mockVisitorRepo, mockTrackRepo, controller := setupTrackTest()

	adminUser := &entity.User{
		ID:       uuid.New(),
		UserName: "admin",
		Name:     "Admin",
		Surname:  "User",
		Role:     entity.UserRoleAdmin,
	}

	visitor := &entity.Visitor{
		Id:      456,
		Name:    "Bob",
		Surname: "Johnson",
	}

	desc := "forgot key"
	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:          1,
		VisitorId:   456,
		Visitor:     visitor,
		AdminID:     &adminUser.ID,
		Description: &desc,
		CreatedAt:   createdAt,
	}

	mockVisitorRepo.On("FindById", int32(456)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(1, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{
		VisitorID:   456,
		Description: &desc,
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ManualTrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-in", response.TrackType)
	assert.Equal(t, visitor.Id, response.Visitor.Id)
	assert.Equal(t, visitor.Name, response.Visitor.Name)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestManualTrackHandler_Success_SignOut(t *testing.T) {
	mockVisitorRepo, mockTrackRepo, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}
	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        2,
		VisitorId: 456,
		Visitor:   visitor,
		AdminID:   &adminUser.ID,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(456)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(2, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ManualTrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-out", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestManualTrackHandler_SignedInTrue(t *testing.T) {
	mockVisitorRepo, mockTrackRepo, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}
	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        1,
		VisitorId: 456,
		SignedIn:  true,
		Visitor:   visitor,
		AdminID:   &adminUser.ID,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(456)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.MatchedBy(func(vt *entity.VisitTrack) bool {
		return vt.SignedIn == true
	})).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(1, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456, SignedIn: true})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestManualTrackHandler_SignedInFalse(t *testing.T) {
	mockVisitorRepo, mockTrackRepo, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}
	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        1,
		VisitorId: 456,
		SignedIn:  false,
		Visitor:   visitor,
		AdminID:   &adminUser.ID,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(456)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.MatchedBy(func(vt *entity.VisitTrack) bool {
		return vt.SignedIn == false
	})).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(2, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456, SignedIn: false})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ManualTrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-out", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestManualTrackHandler_NoAuthContext(t *testing.T) {
	_, _, controller := setupTrackTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestManualTrackHandler_InvalidAuthData(t *testing.T) {
	_, _, controller := setupTrackTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", "not-a-user-struct")

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestManualTrackHandler_InvalidJSON(t *testing.T) {
	_, _, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestManualTrackHandler_MissingVisitorID(t *testing.T) {
	_, _, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(map[string]string{"description": "test"})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestManualTrackHandler_VisitorNotFound(t *testing.T) {
	mockVisitorRepo, _, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}

	mockVisitorRepo.On("FindById", int32(999)).Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 999})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockVisitorRepo.AssertExpectations(t)
}

func TestManualTrackHandler_VisitorNilResult(t *testing.T) {
	mockVisitorRepo, _, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}

	mockVisitorRepo.On("FindById", int32(456)).Return((*entity.Visitor)(nil), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockVisitorRepo.AssertExpectations(t)
}

func TestManualTrackHandler_StoreError(t *testing.T) {
	mockVisitorRepo, mockTrackRepo, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}
	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}

	mockVisitorRepo.On("FindById", int32(456)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(nil, errors.New("store failed"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "store failed", response["error"])

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestManualTrackHandler_CountEventsError(t *testing.T) {
	mockVisitorRepo, mockTrackRepo, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}
	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        1,
		VisitorId: 456,
		Visitor:   visitor,
		AdminID:   &adminUser.ID,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(456)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(0, errors.New("count error"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ManualTrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-in", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestManualTrackHandler_NilDescription(t *testing.T) {
	mockVisitorRepo, mockTrackRepo, controller := setupTrackTest()

	adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}
	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        1,
		VisitorId: 456,
		Visitor:   visitor,
		AdminID:   &adminUser.ID,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(456)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(1, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", adminUser)

	body, _ := json.Marshal(ManualTrackRequest{VisitorID: 456})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/track/visit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.ManualTrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}
