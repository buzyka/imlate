package visitor

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVisitorRepository is a mock implementation of entity.VisitorRepository
type MockVisitorRepository struct {
	mock.Mock
}

func (m *MockVisitorRepository) FindById(id int32) (*entity.Visitor, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Visitor), args.Error(1)
}

func (m *MockVisitorRepository) FindByKey(key string) (*entity.VisitDetails, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VisitDetails), args.Error(1)
}

func (m *MockVisitorRepository) AddKeyToVisitor(visitor *entity.Visitor, key string) error {
	args := m.Called(visitor, key)
	return args.Error(0)
}

func (m *MockVisitorRepository) AddVisitor(visitor *entity.Visitor) error {
	args := m.Called(visitor)
	return args.Error(0)
}

func (m *MockVisitorRepository) GetAll() ([]*entity.Visitor, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Visitor), args.Error(1)
}

func TestAddKeyHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockVisitorRepository)
	controller := &VisitorController{
		VisitorRepository: mockRepo,
	}

	visitor := &entity.Visitor{
		Id:      1,
		Name:    "John",
		Surname: "Doe",
		Grade:   10,
	}

	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("AddKeyToVisitor", visitor, "KEY123").Return(nil)

	request := AddKeyRequest{
		VisitorID:  1,
		VisitorKey: "KEY123",
	}
	body, _ := json.Marshal(request)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/add-key", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := controller.AddKeyHandler()
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Key successfully added", response["message"])

	mockRepo.AssertExpectations(t)
}

func TestAddKeyHandler_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockVisitorRepository)
	controller := &VisitorController{
		VisitorRepository: mockRepo,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/add-key", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := controller.AddKeyHandler()
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAddKeyHandler_VisitorNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockVisitorRepository)
	controller := &VisitorController{
		VisitorRepository: mockRepo,
	}

	mockRepo.On("FindById", int32(999)).Return(&entity.Visitor{}, nil)

	request := AddKeyRequest{
		VisitorID:  999,
		VisitorKey: "KEY123",
	}
	body, _ := json.Marshal(request)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/add-key", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := controller.AddKeyHandler()
	handler(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Visitor not exists", response["error"])

	mockRepo.AssertExpectations(t)
}

func TestAddKeyHandler_FindByIdError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockVisitorRepository)
	controller := &VisitorController{
		VisitorRepository: mockRepo,
	}

	mockRepo.On("FindById", int32(1)).Return(nil, errors.New("database error"))

	request := AddKeyRequest{
		VisitorID:  1,
		VisitorKey: "KEY123",
	}
	body, _ := json.Marshal(request)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/add-key", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := controller.AddKeyHandler()
	handler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "database error", response["error"])

	mockRepo.AssertExpectations(t)
}

func TestAddKeyHandler_AddKeyError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockVisitorRepository)
	controller := &VisitorController{
		VisitorRepository: mockRepo,
	}

	visitor := &entity.Visitor{
		Id:      1,
		Name:    "John",
		Surname: "Doe",
		Grade:   10,
	}

	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("AddKeyToVisitor", visitor, "KEY123").Return(errors.New("key already exists"))

	request := AddKeyRequest{
		VisitorID:  1,
		VisitorKey: "KEY123",
	}
	body, _ := json.Marshal(request)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/add-key", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := controller.AddKeyHandler()
	handler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "key already exists", response["error"])

	mockRepo.AssertExpectations(t)
}
