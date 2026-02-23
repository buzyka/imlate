package adminapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func intPtr(i int) *int { return &i }

func setupVisitorTest() (*providertest.VisitorRepositoryMock, *AdminAPIController) {
	mockRepo := new(providertest.VisitorRepositoryMock)
	api := &usecase.AdminAPI{
		VisitorRepo: mockRepo,
		Config: &config.Config{
			VisitorImageDir:       "/tmp/test-visitor-images",
			VisitorImageURLPrefix: "/assets/img/visitors",
		},
	}
	controller := &AdminAPIController{AdminAPI: api}
	gin.SetMode(gin.TestMode)
	return mockRepo, controller
}

func TestListVisitorsHandler_Success(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/visitors", nil)

	visitors := []*entity.Visitor{
		{Id: 1, Name: "Alice", Surname: "Smith"},
		{Id: 2, Name: "Bob", Surname: "Jones"},
	}
	mockRepo.On("FindAll", []provider.VisitorFilterOption(nil)).Return(visitors, nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"KEY1"}, nil)
	mockRepo.On("FindKeysByVisitorId", int32(2)).Return([]string{}, nil)

	controller.ListVisitorsHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []entity.Visitor
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestListVisitorsHandler_Error(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/visitors", nil)

	mockRepo.On("FindAll", []provider.VisitorFilterOption(nil)).Return(nil, errors.New("db error"))

	controller.ListVisitorsHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetVisitorHandler_Success(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/visitors/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	visitor := &entity.Visitor{Id: 1, Name: "Alice", Surname: "Smith"}
	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"KEY1"}, nil)

	controller.GetVisitorHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var result entity.Visitor
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", result.Name)
	mockRepo.AssertExpectations(t)
}

func TestGetVisitorHandler_InvalidID(t *testing.T) {
	_, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/visitors/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	controller.GetVisitorHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetVisitorHandler_NotFound(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/visitors/999", nil)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mockRepo.On("FindById", int32(999)).Return(&entity.Visitor{}, nil)

	controller.GetVisitorHandler()(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestCreateVisitorHandler_Success(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(CreateVisitorRequest{
		Name:      "Alice",
		Surname:   "Smith",
		IsStudent: true,
		Grade:     intPtr(5),
		Keys:      []string{"KEY1"},
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/visitors", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	mockRepo.On("SaveVisitor", mock.AnythingOfType("*entity.Visitor")).Run(func(args mock.Arguments) {
		v := args.Get(0).(*entity.Visitor)
		v.Id = 10
	}).Return(nil)
	mockRepo.On("AddKeyToVisitor", mock.AnythingOfType("*entity.Visitor"), "KEY1").Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(10)).Return([]string{"KEY1"}, nil)

	controller.CreateVisitorHandler()(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var result entity.Visitor
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, int32(10), result.Id)
	assert.Equal(t, "Alice", result.Name)
	mockRepo.AssertExpectations(t)
}

func TestCreateVisitorHandler_BadRequest(t *testing.T) {
	_, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/visitors", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.CreateVisitorHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateVisitorHandler_Success(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(UpdateVisitorRequest{
		Name:      "Updated",
		Surname:   "Name",
		IsStudent: false,
		Grade:     nil,
		Keys:      []string{"KEY2"},
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/visitors/1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	existing := &entity.Visitor{Id: 1, Name: "Old", Surname: "Name"}
	mockRepo.On("FindById", int32(1)).Return(existing, nil)
	mockRepo.On("SaveVisitor", mock.AnythingOfType("*entity.Visitor")).Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"KEY1"}, nil).Once()
	mockRepo.On("RemoveKeyFromVisitor", int32(1), "KEY1").Return(nil)
	mockRepo.On("AddKeyToVisitor", mock.AnythingOfType("*entity.Visitor"), "KEY2").Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"KEY2"}, nil).Once()

	controller.UpdateVisitorHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var result entity.Visitor
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "Updated", result.Name)
	mockRepo.AssertExpectations(t)
}

func TestUpdateVisitorHandler_InvalidID(t *testing.T) {
	_, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/visitors/abc", bytes.NewBufferString(`{"name":"test"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	controller.UpdateVisitorHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddVisitorKeyHandler_Success(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(AddKeyRequest{Key: "NEWKEY"})
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/visitors/1/key", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	visitor := &entity.Visitor{Id: 1, Name: "Alice"}
	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("AddKeyToVisitor", visitor, "NEWKEY").Return(nil)

	controller.AddVisitorKeyHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAddVisitorKeyHandler_BadRequest(t *testing.T) {
	_, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/visitors/1/key", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	controller.AddVisitorKeyHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveVisitorKeyHandler_Success(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/visitors/1/key/KEY1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}, {Key: "key", Value: "KEY1"}}

	visitor := &entity.Visitor{Id: 1, Name: "Alice"}
	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("RemoveKeyFromVisitor", int32(1), "KEY1").Return(nil)

	controller.RemoveVisitorKeyHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestRemoveVisitorKeyHandler_InvalidID(t *testing.T) {
	_, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/visitors/abc/key/KEY1", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}, {Key: "key", Value: "KEY1"}}

	controller.RemoveVisitorKeyHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteVisitorHandler_Success(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/visitors/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mockRepo.On("DeleteVisitor", int32(1)).Return(nil)

	controller.DeleteVisitorHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestDeleteVisitorHandler_Error(t *testing.T) {
	mockRepo, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/visitors/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mockRepo.On("DeleteVisitor", int32(1)).Return(errors.New("not found"))

	controller.DeleteVisitorHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestDeleteVisitorHandler_InvalidID(t *testing.T) {
	_, controller := setupVisitorTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/visitors/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	controller.DeleteVisitorHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
