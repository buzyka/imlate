package synchroniser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockERPFactory
type MockERPFactory struct {
	mock.Mock
}

func (m *MockERPFactory) NewClient(ctx context.Context) (erp.Client, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(erp.Client), args.Error(1)
}

// MockERPClient
type MockERPClient struct {
	mock.Mock
}

func (m *MockERPClient) GetStudents(page, pageSize int32) (*isams.StudentsResponse, error) {
	args := m.Called(page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*isams.StudentsResponse), args.Error(1)
}

// MockVisitorRepository
type MockVisitorRepository struct {
	mock.Mock
}

func (m *MockVisitorRepository) GetAll() ([]*entity.Visitor, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Visitor), args.Error(1)
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

func TestSyncAllStudents_Success(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
	}

	// Setup expectations
	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return([]*entity.Visitor{}, nil)

	fullName := "John Doe"
	yearGroup := 10
	lastUpdated := "2023-01-01T12:00:00Z"

	student := isams.Student{
		ID:          123,
		SchoolID:    "S123",
		FullName:    &fullName,
		YearGroup:   &yearGroup,
		LastUpdated: &lastUpdated,
	}

	resp := &isams.StudentsResponse{
		Students:   []isams.Student{student},
		TotalPages: 1,
	}

	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(resp, nil)
	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.Surname == "John Doe" && v.Grade == 10
	})).Return(nil)

	// Execute
	err := sync.SyncAllStudents()

	// Assert
	assert.NoError(t, err)
	mockFactory.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestSyncAllStudents_NewClientError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
	}

	mockFactory.On("NewClient", mock.Anything).Return(nil, errors.New("client error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "client error", err.Error())
}

func TestSyncAllStudents_GetAllError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
	}

	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return(nil, errors.New("db error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

func TestSyncAllStudents_GetStudentsError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
	}

	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return([]*entity.Visitor{}, nil)
	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(nil, errors.New("api error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "api error", err.Error())
}

func TestSyncAllStudents_SaveStudentError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
	}

	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return([]*entity.Visitor{}, nil)

	fullName := "John Doe"
	student := isams.Student{
		ID:       123,
		FullName: &fullName,
	}

	resp := &isams.StudentsResponse{
		Students:   []isams.Student{student},
		TotalPages: 1,
	}

	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(resp, nil)
	mockRepo.On("AddVisitor", mock.Anything).Return(errors.New("save error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "save error", err.Error())
}

func TestSyncAllStudents_Pagination(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
	}

	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return([]*entity.Visitor{}, nil)

	fullName1 := "Student 1"
	student1 := isams.Student{ID: 1, FullName: &fullName1}
	resp1 := &isams.StudentsResponse{
		Students:   []isams.Student{student1},
		TotalPages: 2,
	}

	fullName2 := "Student 2"
	student2 := isams.Student{ID: 2, FullName: &fullName2}
	resp2 := &isams.StudentsResponse{
		Students:   []isams.Student{student2},
		TotalPages: 2,
	}

	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(resp1, nil)
	mockClient.On("GetStudents", int32(2), int32(PageSize)).Return(resp2, nil)

	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 1
	})).Return(nil)
	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 2
	})).Return(nil)

	err := sync.SyncAllStudents()

	assert.NoError(t, err)
	mockClient.AssertNumberOfCalls(t, "GetStudents", 2)
}

func TestSaveStudent_IsUpToDate_NoUpdate(t *testing.T) {
	// mockFactory := new(MockERPFactory)
	// mockClient := new(MockERPClient)
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		// ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
	}

	updatedAt, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	existingVisitor := &entity.Visitor{
		Id:        1,
		ErpID:     123,
		UpdatedAt: updatedAt,
	}

	// Pre-populate current visitors
	// mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	// mockRepo.On("GetAll").Return([]*entity.Visitor{existingVisitor}, nil)

	// Setup sync to populate currentVisitors
	// sync.SyncAllStudents() // This will fail at GetStudents but populate currentVisitors
	// Reset mocks to clear calls from SyncAllStudents setup
	// mockFactory = new(MockERPFactory)
	// mockClient = new(MockERPClient)
	// mockRepo = new(MockVisitorRepository)
	// sync.ERPFactory = mockFactory
	// sync.VisitorRepo = mockRepo

	// Manually set currentVisitors for this test
	sync.currentVisitors = []*entity.Visitor{existingVisitor}

	lastUpdatedStr := "2023-01-01T12:00:00Z"
	student := isams.Student{
		ID:          123,
		LastUpdated: &lastUpdatedStr,
	}

	// No AddVisitor call expected because it's up to date
	err := sync.SaveStudent(student)

	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "AddVisitor")
}

func TestSaveStudent_IsUpToDate_UpdateNeeded(t *testing.T) {
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		VisitorRepo: mockRepo,
	}

	oldTime, _ := time.Parse(time.RFC3339, "2022-01-01T12:00:00Z")
	existingVisitor := &entity.Visitor{
		Id:        1,
		ErpID:     123,
		UpdatedAt: oldTime,
	}
	sync.currentVisitors = []*entity.Visitor{existingVisitor}

	newTimeStr := "2023-01-01T12:00:00Z"
	student := isams.Student{
		ID:          123,
		LastUpdated: &newTimeStr,
	}

	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.Id == 1 && v.ErpID == 123
	})).Return(nil)

	err := sync.SaveStudent(student)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSaveStudent_NewVisitor(t *testing.T) {
	mockRepo := new(MockVisitorRepository)

	sync := &StudentSync{
		VisitorRepo: mockRepo,
	}
	sync.currentVisitors = []*entity.Visitor{}

	student := isams.Student{
		ID: 123,
	}

	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.Id == 0
	})).Return(nil)

	err := sync.SaveStudent(student)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSaveStudent_InvalidTimeFormat(t *testing.T) {
	mockRepo := new(MockVisitorRepository)
	sync := &StudentSync{VisitorRepo: mockRepo}
	sync.currentVisitors = []*entity.Visitor{}

	invalidTime := "invalid-time"
	student := isams.Student{
		ID:          123,
		LastUpdated: &invalidTime,
	}

	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.UpdatedAt.IsZero() == false // Should default to Now() in IsUpToDate if zero
	})).Return(nil)

	err := sync.SaveStudent(student)
	assert.NoError(t, err)
}

func TestIsUpToDate_ZeroUpdatedAt_NoUpdate(t *testing.T) {
	sync := &StudentSync{}
	existingVisitor := &entity.Visitor{
		Id:    1,
		ErpID: 123,
	}
	sync.currentVisitors = []*entity.Visitor{existingVisitor}

	newVisitor := &entity.Visitor{
		ErpID: 123,
	}

	// If new visitor has zero UpdatedAt, it should return true (up to date)
	// based on the logic: case newVisitor.UpdatedAt.IsZero(): return true
	result := sync.IsUpToDate(newVisitor)
	assert.True(t, result)
	assert.Equal(t, int32(1), newVisitor.Id)
}

func TestIsUpToDate_NewVisitor_SetsUpdatedAt(t *testing.T) {
	sync := &StudentSync{}
	sync.currentVisitors = []*entity.Visitor{}

	newVisitor := &entity.Visitor{
		ErpID: 123,
	}

	result := sync.IsUpToDate(newVisitor)
	assert.False(t, result)
	assert.False(t, newVisitor.UpdatedAt.IsZero())
}
