package tracker

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
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func intPtr(i int) *int   { return &i }
func strPtr(s string) *string { return &s }

func TestTrackHandler_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{
		Id:      123,
		Name:    "Jane",
		Surname: "Smith",
		Grade:   intPtr(11),
		Image:   "/assets/img/teachers/2.jpg",
	}

	requestData := Request{
		VisitorID: 123,
		VisitKey:  "KEY456",
		SignedIn:  true,
	}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        1,
		VisitorId: 123,
		VisitKey:  strPtr("KEY456"),
		SignedIn:  true,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(123)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/track", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.TrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "tracked", response["message"])
	assert.Equal(t, float64(123), response["id"])
	assert.Equal(t, "KEY456", response["vk"])
	assert.Equal(t, true, response["tr"])

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestTrackHandler_InvalidJSON(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	// Create test request with invalid JSON
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest(http.MethodPost, "/track", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.TrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestTrackHandler_VisitorNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	requestData := Request{
		VisitorID: 999,
		VisitKey:  "KEY789",
		SignedIn:  false,
	}

	mockVisitorRepo.On("FindById", int32(999)).Return(nil, errors.New("not found"))

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/track", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.TrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Visitor not exists", response["error"])

	mockVisitorRepo.AssertExpectations(t)
}

func TestTrackHandler_StoreError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{
		Id:      123,
		Name:    "Jane",
		Surname: "Smith",
	}

	requestData := Request{
		VisitorID: 123,
		VisitKey:  "KEY456",
		SignedIn:  true,
	}

	mockVisitorRepo.On("FindById", int32(123)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(nil, errors.New("database error"))

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/track", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.TrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "database error", response["error"])

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

// admin_id in the tracker Request identifies the admin user with role "terminal"
// that operates the physical trackpoint device. It is stored in the track.admin_id column.
func TestTrackHandler_WithAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{Id: 123, Name: "Jane", Surname: "Smith"}
	terminalAdminID := uuid.New().String()

	requestData := Request{
		VisitorID: 123,
		VisitKey:  "KEY456",
		SignedIn:  true,
		AdminID:   &terminalAdminID,
	}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        1,
		VisitorId: 123,
		VisitKey:  strPtr("KEY456"),
		SignedIn:  true,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(123)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.MatchedBy(func(vt *entity.VisitTrack) bool {
		return vt.AdminID != nil && vt.AdminID.String() == terminalAdminID
	})).Return(storedTrack, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/track", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	controller.TrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestTrackHandler_WithInvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	invalidID := "not-a-uuid"

	requestData := Request{
		VisitorID: 123,
		VisitKey:  "KEY456",
		SignedIn:  true,
		AdminID:   &invalidID,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/track", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	controller.TrackHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid admin_id")
}

func TestTrackHandler_WithNilAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{Id: 123, Name: "Jane", Surname: "Smith"}

	requestData := Request{
		VisitorID: 123,
		VisitKey:  "KEY456",
		SignedIn:  true,
	}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        1,
		VisitorId: 123,
		VisitKey:  strPtr("KEY456"),
		SignedIn:  true,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindById", int32(123)).Return(visitor, nil)
	mockTrackRepo.On("Store", mock.MatchedBy(func(vt *entity.VisitTrack) bool {
		return vt.AdminID == nil
	})).Return(storedTrack, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/track", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	controller.TrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{
		Id:      456,
		Name:    "Bob",
		Surname: "Johnson",
		Grade:   intPtr(9),
		Image:   "/assets/img/teachers/3.jpg",
	}

	visitDetails := &entity.VisitDetails{
		Visitor: visitor,
		Key:     "FIND123",
	}

	requestData := Request{
		VisitKey: "FIND123",
		SignedIn: true,
	}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        2,
		VisitorId: 456,
		VisitKey:  strPtr("FIND123"),
		SignedIn:  true,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindByKey", "FIND123").Return(visitDetails, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.MatchedBy(func(t time.Time) bool { return true })).Return(1, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.FindAndTrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response TrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, visitor.Id, response.Visitor.Id)
	assert.Equal(t, "sign-in", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_SignOut(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{
		Id:      456,
		Name:    "Bob",
		Surname: "Johnson",
		Grade:   intPtr(9),
	}

	visitDetails := &entity.VisitDetails{
		Visitor: visitor,
		Key:     "FIND123",
	}

	requestData := Request{
		VisitKey: "FIND123",
		SignedIn: false,
	}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        2,
		VisitorId: 456,
		VisitKey:  strPtr("FIND123"),
		SignedIn:  false,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindByKey", "FIND123").Return(visitDetails, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(2, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.FindAndTrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response TrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-out", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_InvalidJSON(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	// Create test request with invalid JSON
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.FindAndTrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFindAndTrackHandler_VisitorNotFoundByKey(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	requestData := Request{
		VisitKey: "NOTFOUND",
		SignedIn: true,
	}

	mockVisitorRepo.On("FindByKey", "NOTFOUND").Return((*entity.VisitDetails)(nil), errors.New("not found"))

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.FindAndTrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Visitor not exists", response["error"])

	mockVisitorRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_NilVisitor(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitDetails := &entity.VisitDetails{
		Visitor: nil,
		Key:     "EMPTY",
	}

	requestData := Request{
		VisitKey: "EMPTY",
		SignedIn: true,
	}

	mockVisitorRepo.On("FindByKey", "EMPTY").Return(visitDetails, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.FindAndTrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	mockVisitorRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_StoreError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{
		Id:      456,
		Name:    "Bob",
		Surname: "Johnson",
	}

	visitDetails := &entity.VisitDetails{
		Visitor: visitor,
		Key:     "FIND123",
	}

	requestData := Request{
		VisitKey: "FIND123",
		SignedIn: true,
	}

	mockVisitorRepo.On("FindByKey", "FIND123").Return(visitDetails, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(nil, errors.New("store failed"))

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.FindAndTrackHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "store failed", response["error"])

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_CountEventsError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{
		Id:      456,
		Name:    "Bob",
		Surname: "Johnson",
	}

	visitDetails := &entity.VisitDetails{
		Visitor: visitor,
		Key:     "FIND123",
	}

	requestData := Request{
		VisitKey: "FIND123",
		SignedIn: true,
	}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        2,
		VisitorId: 456,
		VisitKey:  strPtr("FIND123"),
		SignedIn:  true,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindByKey", "FIND123").Return(visitDetails, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(0, errors.New("count error"))

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Execute
	handler := controller.FindAndTrackHandler()
	handler(c)

	// Assert - should still succeed with default "sign-in" type
	assert.Equal(t, http.StatusOK, w.Code)

	var response TrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-in", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_CountFromStartOfDay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}
	visitDetails := &entity.VisitDetails{Visitor: visitor, Key: "FIND123"}

	createdAt := time.Date(2026, 3, 18, 14, 30, 45, 0, time.Local)
	storedTrack := &entity.VisitTrack{
		Id:        2,
		VisitorId: 456,
		VisitKey:  strPtr("FIND123"),
		SignedIn:  true,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindByKey", "FIND123").Return(visitDetails, nil)
	mockTrackRepo.On("Store", mock.AnythingOfType("*entity.VisitTrack")).Return(storedTrack, nil)
	mockTrackRepo.
		On("CountEventsByVisitorIdSince", int32(456), mock.MatchedBy(func(d time.Time) bool {
			return d.Year() == 2026 && d.Month() == time.March && d.Day() == 18 && d.Hour() == 0 && d.Minute() == 0 && d.Second() == 0
		})).
		Return(1, nil)

	requestData := Request{VisitKey: "FIND123", SignedIn: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler := controller.FindAndTrackHandler()
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response TrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-in", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_WithAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	visitor := &entity.Visitor{Id: 456, Name: "Bob", Surname: "Johnson"}
	visitDetails := &entity.VisitDetails{Visitor: visitor, Key: "FIND123"}
	terminalAdminID := uuid.New().String()

	requestData := Request{
		VisitKey: "FIND123",
		SignedIn: true,
		AdminID:  &terminalAdminID,
	}

	createdAt := time.Now()
	storedTrack := &entity.VisitTrack{
		Id:        2,
		VisitorId: 456,
		VisitKey:  strPtr("FIND123"),
		SignedIn:  true,
		Visitor:   visitor,
		CreatedAt: createdAt,
	}

	mockVisitorRepo.On("FindByKey", "FIND123").Return(visitDetails, nil)
	mockTrackRepo.On("Store", mock.MatchedBy(func(vt *entity.VisitTrack) bool {
		return vt.AdminID != nil && vt.AdminID.String() == terminalAdminID
	})).Return(storedTrack, nil)
	mockTrackRepo.On("CountEventsByVisitorIdSince", int32(456), mock.AnythingOfType("time.Time")).Return(1, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	controller.FindAndTrackHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response TrackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sign-in", response.TrackType)

	mockVisitorRepo.AssertExpectations(t)
	mockTrackRepo.AssertExpectations(t)
}

func TestFindAndTrackHandler_WithInvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVisitorRepo := new(providertest.VisitorRepositoryMock)
	mockTrackRepo := new(providertest.VisitorTrackRepositoryMock)
	controller := &TrackerController{
		VisitorRepository: mockVisitorRepo,
		TrackRepository:   mockTrackRepo,
	}

	invalidID := "not-a-uuid"

	requestData := Request{
		VisitKey: "FIND123",
		SignedIn: true,
		AdminID:  &invalidID,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest(http.MethodPost, "/findtrack", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	controller.FindAndTrackHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid admin_id")
}

func TestChangeTimeHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &TrackerController{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/change-time", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler := controller.ChangeTimeHandler()
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestChangeTimeHandler_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &TrackerController{}

	payload := map[string]string{"time": "8:05"}
	jsonData, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/change-time", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler := controller.ChangeTimeHandler()
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid time format", response["error"])
}

func TestChangeTimeHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &TrackerController{}

	payload := map[string]string{"time": "08:05"}
	jsonData, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/change-time", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler := controller.ChangeTimeHandler()
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Changed time to: 8:5", response["message"])
}

func TestFakeTrackerService_Track(t *testing.T) {
	// Test the fake tracker service
	service := FakeTrackerService{}
	err := service.Track("test-id")

	assert.NoError(t, err)
}
