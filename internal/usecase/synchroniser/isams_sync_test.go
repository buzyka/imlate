package synchroniser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
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

func (m *MockERPClient) GetYearGroupDivisions(yearGroupID int32) (*isams.YearGroupsDivisionsResponse, error) {
	args := m.Called(yearGroupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*isams.YearGroupsDivisionsResponse), args.Error(1)
}

func (m *MockERPClient) GetCurrentRegistrationPeriodsForDivision(divisionID int32) (*isams.RegistrationPeriodsResponse, error) {
	args := m.Called(divisionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*isams.RegistrationPeriodsResponse), args.Error(1)
}

func (m *MockERPClient) GetRegistrationStatusForStudent(studentSchoolID string, periodID int32) (*isams.RegistrationStatus, error) {
	args := m.Called(studentSchoolID, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*isams.RegistrationStatus), args.Error(1)
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

	divisionsResp := &isams.YearGroupsDivisionsResponse{
		Divisions: []isams.Division{
			{ID: 1, Name: "Div 1"},
		},
	}

	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(resp, nil)
	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(divisionsResp, nil)
	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.Surname == "John Doe" && v.Grade == 10 && len(v.ErpDivisions) == 1 && v.ErpDivisions[0] == 1
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
	yearGroup := 10
	student := isams.Student{
		ID:       123,
		FullName: &fullName,
		YearGroup: &yearGroup,
	}

	resp := &isams.StudentsResponse{
		Students:   []isams.Student{student},
		TotalPages: 1,
	}

	divisionsResp := &isams.YearGroupsDivisionsResponse{
		Divisions: []isams.Division{},
	}

	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(resp, nil)
	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(divisionsResp, nil)
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
	yearGroup1 := 10
	student1 := isams.Student{ID: 1, FullName: &fullName1, YearGroup: &yearGroup1}
	resp1 := &isams.StudentsResponse{
		Students:   []isams.Student{student1},
		TotalPages: 2,
	}

	fullName2 := "Student 2"
	yearGroup2 := 11
	student2 := isams.Student{ID: 2, FullName: &fullName2, YearGroup: &yearGroup2}
	resp2 := &isams.StudentsResponse{
		Students:   []isams.Student{student2},
		TotalPages: 2,
	}

	divisionsResp1 := &isams.YearGroupsDivisionsResponse{Divisions: []isams.Division{}}
	divisionsResp2 := &isams.YearGroupsDivisionsResponse{Divisions: []isams.Division{}}

	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(resp1, nil)
	mockClient.On("GetStudents", int32(2), int32(PageSize)).Return(resp2, nil)
	mockClient.On("GetYearGroupDivisions", int32(yearGroup1)).Return(divisionsResp1, nil)
	mockClient.On("GetYearGroupDivisions", int32(yearGroup2)).Return(divisionsResp2, nil)

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
	mockRepo := new(MockVisitorRepository)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo: mockRepo,
		currentClient: mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}

	updatedAt, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	yearGroup := 10
	existingVisitor := &entity.Visitor{
		Id:        1,
		ErpID:     123,
		UpdatedAt: updatedAt,
		ErpYearGroupID: int32(yearGroup),
		ErpDivisions: []int32{},
	}

	sync.currentVisitors = []*entity.Visitor{existingVisitor}

	lastUpdatedStr := "2023-01-01T12:00:00Z"
	
	student := isams.Student{
		ID:          123,
		LastUpdated: &lastUpdatedStr,
		YearGroup:   &yearGroup,
	}

	divisionsResp := &isams.YearGroupsDivisionsResponse{Divisions: []isams.Division{}}
	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(divisionsResp, nil)

	// No AddVisitor call expected because it's up to date
	err := sync.SaveStudent(student)

	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "AddVisitor")
}

func TestSaveStudent_IsUpToDate_UpdateNeeded(t *testing.T) {
	mockRepo := new(MockVisitorRepository)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo: mockRepo,
		currentClient: mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}

	oldTime, _ := time.Parse(time.RFC3339, "2022-01-01T12:00:00Z")
	existingVisitor := &entity.Visitor{
		Id:        1,
		ErpID:     123,
		UpdatedAt: oldTime,
	}
	sync.currentVisitors = []*entity.Visitor{existingVisitor}

	newTimeStr := "2023-01-01T12:00:00Z"
	yearGroup := 10
	student := isams.Student{
		ID:          123,
		LastUpdated: &newTimeStr,
		YearGroup:   &yearGroup,
	}

	divisionsResp := &isams.YearGroupsDivisionsResponse{Divisions: []isams.Division{}}
	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(divisionsResp, nil)

	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.Id == 1 && v.ErpID == 123
	})).Return(nil)

	err := sync.SaveStudent(student)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSaveStudent_NewVisitor(t *testing.T) {
	mockRepo := new(MockVisitorRepository)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo: mockRepo,
		currentClient: mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}
	sync.currentVisitors = []*entity.Visitor{}

	yearGroup := 10
	student := isams.Student{
		ID: 123,
		YearGroup: &yearGroup,
	}

	divisionsResp := &isams.YearGroupsDivisionsResponse{Divisions: []isams.Division{}}
	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(divisionsResp, nil)

	mockRepo.On("AddVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.Id == 0
	})).Return(nil)

	err := sync.SaveStudent(student)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSaveStudent_InvalidTimeFormat(t *testing.T) {
	mockRepo := new(MockVisitorRepository)
	mockClient := new(MockERPClient)
	sync := &StudentSync{
		VisitorRepo: mockRepo,
		currentClient: mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}
	sync.currentVisitors = []*entity.Visitor{}

	invalidTime := "invalid-time"
	yearGroup := 10
	student := isams.Student{
		ID:          123,
		LastUpdated: &invalidTime,
		YearGroup:   &yearGroup,
	}

	divisionsResp := &isams.YearGroupsDivisionsResponse{Divisions: []isams.Division{}}
	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(divisionsResp, nil)

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

func TestSaveStudent_GetDivisionsError(t *testing.T) {
	mockRepo := new(MockVisitorRepository)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo: mockRepo,
		currentClient: mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}

	yearGroup := 10
	student := isams.Student{
		ID: 123,
		YearGroup: &yearGroup,
	}

	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(nil, errors.New("division error"))

	err := sync.SaveStudent(student)

	assert.Error(t, err)
	assert.Equal(t, "division error", err.Error())
}

func TestGetDivisionsByYearGroup_Cache(t *testing.T) {
	mockClient := new(MockERPClient)
	sync := &StudentSync{
		currentClient: mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}

	yearGroup := int32(10)
	divisionsResp := &isams.YearGroupsDivisionsResponse{
		Divisions: []isams.Division{
			{ID: 1, Name: "Div 1"},
		},
	}

	// First call - should hit API
	mockClient.On("GetYearGroupDivisions", yearGroup).Return(divisionsResp, nil).Once()

	divs, err := sync.getDivisionsByYearGroup(yearGroup)
	assert.NoError(t, err)
	assert.Equal(t, []int32{1}, divs)

	// Second call - should hit cache (no mock call expected)
	divs, err = sync.getDivisionsByYearGroup(yearGroup)
	assert.NoError(t, err)
	assert.Equal(t, []int32{1}, divs)

	mockClient.AssertExpectations(t)
}

func TestGetDivisionsByYearGroup_Error(t *testing.T) {
	mockClient := new(MockERPClient)
	sync := &StudentSync{
		currentClient: mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}

	yearGroup := int32(10)
	mockClient.On("GetYearGroupDivisions", yearGroup).Return(nil, errors.New("api error"))

	divs, err := sync.getDivisionsByYearGroup(yearGroup)
	assert.Error(t, err)
	assert.Nil(t, divs)
	assert.Equal(t, "api error", err.Error())
}
