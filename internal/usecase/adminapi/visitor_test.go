package adminapi

import (
	"errors"
	"testing"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func intPtr(i int) *int { return &i }

func newVisitorTestAPI() (*AdminAPI, *providertest.VisitorRepositoryMock) {
	mockRepo := new(providertest.VisitorRepositoryMock)
	api := &AdminAPI{
		VisitorRepo: mockRepo,
		Config: &config.Config{
			VisitorImageDir:       "/tmp/test-visitor-images",
			VisitorImageURLPrefix: "/assets/img/visitors",
		},
	}
	return api, mockRepo
}

func TestGetAllVisitors_Success(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	visitors := []*entity.Visitor{
		{Id: 1, Name: "Alice", Surname: "Smith", ErpID: 1001, ErpSchoolID: "S1001"},
		{Id: 2, Name: "Bob", Surname: "Jones"},
	}
	mockRepo.On("FindAll", []provider.VisitorFilterOption(nil)).Return(visitors, nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"KEY1"}, nil)
	mockRepo.On("FindKeysByVisitorId", int32(2)).Return([]string{"KEY2", "KEY3"}, nil)

	result, err := api.GetAllVisitors()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, []string{"KEY1"}, result[0].Keys)
	assert.Equal(t, []string{"KEY2", "KEY3"}, result[1].Keys)
	assert.True(t, result[0].ImportedFromISAMS)
	assert.False(t, result[1].ImportedFromISAMS)
	mockRepo.AssertExpectations(t)
}

func TestGetAllVisitors_Error(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("FindAll", []provider.VisitorFilterOption(nil)).Return(nil, errors.New("db error"))

	result, err := api.GetAllVisitors()

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestGetVisitor_Success(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	visitor := &entity.Visitor{Id: 1, Name: "Alice", Surname: "Smith", ErpID: 1001, ErpSchoolID: "S1001"}
	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"KEY1"}, nil)

	result, err := api.GetVisitor(1)

	assert.NoError(t, err)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, []string{"KEY1"}, result.Keys)
	assert.True(t, result.ImportedFromISAMS)
	mockRepo.AssertExpectations(t)
}

func TestGetVisitor_NotFound(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("FindById", int32(999)).Return(&entity.Visitor{}, nil)

	result, err := api.GetVisitor(999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "visitor not found")
	mockRepo.AssertExpectations(t)
}

func TestCreateVisitor_Success(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("SaveVisitor", mock.AnythingOfType("*entity.Visitor")).Run(func(args mock.Arguments) {
		v := args.Get(0).(*entity.Visitor)
		v.Id = 10
	}).Return(nil)
	mockRepo.On("AddKeyToVisitor", mock.AnythingOfType("*entity.Visitor"), "KEY1").Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(10)).Return([]string{"KEY1"}, nil)

	result, err := api.CreateVisitor("Alice", "Smith", true, intPtr(5), "", []string{" key1 "})

	assert.NoError(t, err)
	assert.Equal(t, int32(10), result.Id)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, "Smith", result.Surname)
	assert.True(t, result.IsStudent)
	assert.Equal(t, intPtr(5), result.Grade)
	assert.Equal(t, []string{"KEY1"}, result.Keys)
	assert.False(t, result.ImportedFromISAMS)
	mockRepo.AssertExpectations(t)
}

func TestCreateVisitor_IgnoreEmptyTrimmedKeys(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("SaveVisitor", mock.AnythingOfType("*entity.Visitor")).Run(func(args mock.Arguments) {
		v := args.Get(0).(*entity.Visitor)
		v.Id = 12
	}).Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(12)).Return([]string{}, nil)

	result, err := api.CreateVisitor("Alice", "Smith", false, nil, "", []string{"   ", ""})

	assert.NoError(t, err)
	assert.Equal(t, int32(12), result.Id)
	assert.False(t, result.ImportedFromISAMS)
	mockRepo.AssertExpectations(t)
}

func TestCreateVisitor_SaveError(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("SaveVisitor", mock.AnythingOfType("*entity.Visitor")).Return(errors.New("db error"))

	result, err := api.CreateVisitor("Alice", "Smith", false, nil, "", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestCreateVisitor_NoKeys(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("SaveVisitor", mock.AnythingOfType("*entity.Visitor")).Run(func(args mock.Arguments) {
		v := args.Get(0).(*entity.Visitor)
		v.Id = 11
	}).Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(11)).Return([]string{}, nil)

	result, err := api.CreateVisitor("Bob", "Jones", false, nil, "", nil)

	assert.NoError(t, err)
	assert.Equal(t, int32(11), result.Id)
	assert.Equal(t, []string{}, result.Keys)
	assert.False(t, result.ImportedFromISAMS)
	mockRepo.AssertExpectations(t)
}

func TestUpdateVisitor_Success(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	existing := &entity.Visitor{Id: 1, Name: "Old", Surname: "Name"}
	mockRepo.On("FindById", int32(1)).Return(existing, nil)
	mockRepo.On("SaveVisitor", mock.AnythingOfType("*entity.Visitor")).Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"OLD_KEY"}, nil).Once()
	mockRepo.On("RemoveKeyFromVisitor", int32(1), "OLD_KEY").Return(nil)
	mockRepo.On("AddKeyToVisitor", mock.AnythingOfType("*entity.Visitor"), "NEW_KEY").Return(nil)
	mockRepo.On("FindKeysByVisitorId", int32(1)).Return([]string{"NEW_KEY"}, nil).Once()

	result, err := api.UpdateVisitor(1, "New", "Name", true, intPtr(3), "", []string{"NEW_KEY"})

	assert.NoError(t, err)
	assert.Equal(t, "New", result.Name)
	assert.Equal(t, true, result.IsStudent)
	assert.Equal(t, intPtr(3), result.Grade)
	assert.Equal(t, []string{"NEW_KEY"}, result.Keys)
	assert.False(t, result.ImportedFromISAMS)
	mockRepo.AssertExpectations(t)
}

func TestUpdateVisitor_NotFound(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("FindById", int32(999)).Return(&entity.Visitor{}, nil)

	result, err := api.UpdateVisitor(999, "Name", "Sur", false, nil, "", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "visitor not found")
	mockRepo.AssertExpectations(t)
}

func TestAddVisitorKey_Success(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	visitor := &entity.Visitor{Id: 1, Name: "Alice"}
	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("AddKeyToVisitor", visitor, "NEWKEY").Return(nil)

	err := api.AddVisitorKey(1, " newKey ")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAddVisitorKey_EmptyKey(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	visitor := &entity.Visitor{Id: 1, Name: "Alice"}
	mockRepo.On("FindById", int32(1)).Return(visitor, nil)

	err := api.AddVisitorKey(1, "  ")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")
	mockRepo.AssertExpectations(t)
}

func TestAddVisitorKey_VisitorNotFound(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("FindById", int32(999)).Return(&entity.Visitor{}, nil)

	err := api.AddVisitorKey(999, "KEY1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "visitor not found")
	mockRepo.AssertExpectations(t)
}

func TestRemoveVisitorKey_Success(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	visitor := &entity.Visitor{Id: 1, Name: "Alice"}
	mockRepo.On("FindById", int32(1)).Return(visitor, nil)
	mockRepo.On("RemoveKeyFromVisitor", int32(1), "KEY1").Return(nil)

	err := api.RemoveVisitorKey(1, "KEY1")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRemoveVisitorKey_VisitorNotFound(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("FindById", int32(999)).Return(&entity.Visitor{}, nil)

	err := api.RemoveVisitorKey(999, "KEY1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "visitor not found")
	mockRepo.AssertExpectations(t)
}

func TestDeleteVisitor_Success(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("DeleteVisitor", int32(1)).Return(nil)

	err := api.DeleteVisitor(1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteVisitor_Error(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("DeleteVisitor", int32(999)).Return(errors.New("visitor not found or already deleted"))

	err := api.DeleteVisitor(999)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUploadVisitorImage_VisitorNotFound(t *testing.T) {
	api, mockRepo := newVisitorTestAPI()

	mockRepo.On("FindById", int32(999)).Return(&entity.Visitor{}, nil)

	result, err := api.UploadVisitorImage(999, "photo.jpg", []byte("data"))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "visitor not found")
	mockRepo.AssertExpectations(t)
}

func TestNewVisitorResponse(t *testing.T) {
	assert.Nil(t, newVisitorResponse(nil))

	grade := 9
	visitor := &entity.Visitor{
		Id:           7,
		Name:         "Alice",
		Surname:      "Smith",
		Email:        "alice@example.com",
		IsStudent:    true,
		Grade:        &grade,
		Image:        "/img/alice.jpg",
		ErpID:        555,
		ErpSchoolID:  "S555",
		ErpDivisions: []int32{1, 2},
		Keys:         []string{"KEY1", "KEY2"},
	}

	result := newVisitorResponse(visitor)
	list := newVisitorResponses([]*entity.Visitor{visitor})

	assert.NotNil(t, result)
	assert.True(t, result.ImportedFromISAMS)
	assert.Equal(t, []string{"KEY1", "KEY2"}, result.Keys)
	assert.Equal(t, []int32{1, 2}, result.ErpDivisions)
	assert.Len(t, list, 1)
	assert.Equal(t, result, list[0])
}
