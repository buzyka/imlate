package search

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSearchHandler_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockRepo := new(providertest.VisitorRepositoryMock)
	controller := SearchController{
		StudentRepository: mockRepo,
	}

	visitor := &entity.Visitor{
		Id:      1,
		Name:    "John",
		Surname: "Doe",
		Grade:   10,
		Image:   "/assets/img/teachers/1.jpg",
	}

	visitDetails := &entity.VisitDetails{
		Visitor: visitor,
		Key:     "TEST123",
	}

	mockRepo.On("FindByKey", "TEST123").Return(visitDetails, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "TEST123"}}
	req, _ := http.NewRequest(http.MethodGet, "/search/TEST123", nil)
	c.Request = req

	// Execute
	handler := controller.SearchHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.VisitDetails
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, visitor.Id, response.Visitor.Id)
	assert.Equal(t, visitor.Name, response.Visitor.Name)
	assert.Equal(t, visitor.Surname, response.Visitor.Surname)
	assert.Equal(t, "TEST123", response.Key)

	mockRepo.AssertExpectations(t)
}

func TestSearchHandler_VisitorNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockRepo := new(providertest.VisitorRepositoryMock)
	controller := SearchController{
		StudentRepository: mockRepo,
	}

	// Return an empty VisitDetails with explicit nil Visitor
	visitDetails := &entity.VisitDetails{
		Visitor: nil,
		Key:     "",
	}

	// Add assertion to verify the mock setup
	assert.Nil(t, visitDetails.Visitor, "Visitor should be nil in test setup")

	mockRepo.On("FindByKey", "NOTFOUND").Return(visitDetails, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "NOTFOUND"}}
	req, _ := http.NewRequest(http.MethodGet, "/search/NOTFOUND", nil)
	c.Request = req

	// Execute
	handler := controller.SearchHandler()
	handler(c)

	// Assert
	t.Logf("Response code: %d, body: '%s', mock called: %v", w.Code, w.Body.String(), mockRepo.AssertNumberOfCalls(t, "FindByKey", 1))

	// The test should return 404 when visitor is not found
	assert.Equal(t, http.StatusNotFound, c.Writer.Status())
	mockRepo.AssertExpectations(t)
}

func TestSearchHandler_InternalError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockRepo := new(providertest.VisitorRepositoryMock)
	controller := SearchController{
		StudentRepository: mockRepo,
	}

	expectedError := errors.New("database connection error")
	mockRepo.On("FindByKey", "ERROR123").Return(nil, expectedError)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "ERROR123"}}
	req, _ := http.NewRequest(http.MethodGet, "/search/ERROR123", nil)
	c.Request = req

	// Execute
	handler := controller.SearchHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedError.Error(), response["error"])

	mockRepo.AssertExpectations(t)
}

func TestSearchHandler_EmptyKey(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockRepo := new(providertest.VisitorRepositoryMock)
	controller := SearchController{
		StudentRepository: mockRepo,
	}

	visitDetails := &entity.VisitDetails{
		Visitor: nil,
	}

	mockRepo.On("FindByKey", "").Return(visitDetails, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: ""}}
	req, _ := http.NewRequest(http.MethodGet, "/search/", nil)
	c.Request = req

	// Execute
	handler := controller.SearchHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, c.Writer.Status())
	assert.Empty(t, w.Body.String())
	mockRepo.AssertExpectations(t)
}

func TestSearchHandler_NilVisitorInDetails(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockRepo := new(providertest.VisitorRepositoryMock)
	controller := SearchController{
		StudentRepository: mockRepo,
	}

	visitDetails := &entity.VisitDetails{
		Visitor: nil,
	}

	mockRepo.On("FindByKey", "KEY123").Return(visitDetails, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "KEY123"}}
	req, _ := http.NewRequest(http.MethodGet, "/search/KEY123", nil)
	c.Request = req

	// Execute
	handler := controller.SearchHandler()
	handler(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, c.Writer.Status())
	assert.Empty(t, w.Body.String())
	mockRepo.AssertExpectations(t)
}
