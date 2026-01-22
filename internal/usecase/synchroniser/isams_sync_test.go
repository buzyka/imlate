package synchroniser

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

//go:embed _fixtures/single_student.json
var testSingleStudentResponse []byte

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

func (m *MockERPClient) GetRegistrationAbsenceCodes() (*isams.RegistrationAbsenceCodesResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*isams.RegistrationAbsenceCodesResponse), args.Error(1)
}

func (m *MockERPClient) GetRegistrationPresentCodes() (*isams.RegistrationPresentCodeResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*isams.RegistrationPresentCodeResponse), args.Error(1)
}

func (m *MockERPClient) PutRegistration(schoolID string, periodID int32, request isams.RegistrationStatusRequest) error {
	args := m.Called(schoolID, periodID, request)
	return args.Error(0)
}

func (m *MockERPClient) GetStudentPhoto(schoolID string) (*isams.StudentPhotoResponse, error) {
	args := m.Called(schoolID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*isams.StudentPhotoResponse), args.Error(1)
}

func TestCleanUpSyncSession(t *testing.T) {
	t.Parallel()
	erpClient := new(MockERPClient)
	ctx := context.Background()
	yg := make(map[int32][]int32)
	yg[1] = []int32{1}

	s := &StudentSync{
		currentClient:      erpClient,
		ctx:                ctx,
		yearGroupDivisions: yg,
	}
	s.cleanUpSyncSession()

	assert.Nil(t, s.currentClient)
	assert.Nil(t, s.ctx)
	assert.Nil(t, s.yearGroupDivisions)
}

func TestStartSyncSession_Error(t *testing.T) {
	mockFactory := new(MockERPFactory)
	logger := zap.NewNop().Sugar()
	sync := &StudentSync{
		ERPFactory: mockFactory,
		Logger:     logger,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(nil, errors.New("client error"))

	err := sync.startSyncSession(context.Background())

	assert.Error(t, err)
	assert.Nil(t, sync.currentClient)
	mockFactory.AssertExpectations(t)
}

func TestStartSyncSession_Success(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	sync := &StudentSync{
		ERPFactory: mockFactory,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	err := sync.startSyncSession(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, mockClient, sync.currentClient)
	mockFactory.AssertExpectations(t)
}

func TestSyncAllStudents_Success(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
	}

	// Setup expectations
	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)
	mockRepo.On("GetAll").Return([]*entity.Visitor{}, nil)

	surname := "Doe"
	forname := "John"
	yearGroup := 10
	lastUpdated := "2023-01-01T12:00:00Z"

	student := isams.Student{
		ID:          123,
		SchoolID:    "S123",
		Forename:    &forname,
		Surname:     &surname,
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
	mockRepo.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.Surname == "Doe" && v.Name == "John" && v.Grade == 10 && len(v.ErpDivisions) == 1 && v.ErpDivisions[0] == 1
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
	mockRepo := new(providertest.VisitorRepositoryMock)
	logger, logBuffer := util.NewTestLogger()

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      logger,
	}

	mockFactory.On("NewClient", mock.Anything).Return(nil, errors.New("client error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "client error", err.Error())
	assert.Contains(t, logBuffer.String(), "create ERP client failed")
}

func TestSyncAllStudents_GetAllError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)
	logger, logBuffer := util.NewTestLogger()

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      logger,
	}

	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return(nil, errors.New("db error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
	assert.Contains(t, logBuffer.String(), "sync all students: get all visitors failed")
}

func TestSyncAllStudents_GetStudentsError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)
	logger, logBuffer := util.NewTestLogger()

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      logger,
	}

	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return([]*entity.Visitor{}, nil)
	mockClient.On("GetStudents", int32(1), int32(PageSize)).Return(nil, errors.New("api error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "api error", err.Error())
	buffStr := logBuffer.String()
	assert.Contains(t, buffStr, "sync all students: get students step failed")
	assert.Contains(t, buffStr, "pageNumber")
	assert.Contains(t, buffStr, "pageSize")
}

func TestSyncAllStudents_SaveStudentError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)
	logger, logBuffer := util.NewTestLogger()

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      logger,
	}

	mockFactory.On("NewClient", mock.Anything).Return(mockClient, nil)
	mockRepo.On("GetAll").Return([]*entity.Visitor{}, nil)

	fullName := "John Doe"
	yearGroup := 10
	student := isams.Student{
		ID:        123,
		FullName:  &fullName,
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
	mockRepo.On("SaveVisitor", mock.Anything).Return(errors.New("save error"))

	err := sync.SyncAllStudents()

	assert.Error(t, err)
	assert.Equal(t, "save error", err.Error())
	buffStr := logBuffer.String()
	assert.Contains(t, buffStr, "sync all students: save student step failed")
	assert.Contains(t, buffStr, `"studentID": 123`)
}

func TestSyncAllStudents_Pagination(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
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

	mockRepo.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 1
	})).Return(nil)
	mockRepo.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 2
	})).Return(nil)

	err := sync.SyncAllStudents()

	assert.NoError(t, err)
	mockClient.AssertNumberOfCalls(t, "GetStudents", 2)
}

func TestSaveStudent_IsUpToDate_NoUpdate(t *testing.T) {
	mockRepo := new(providertest.VisitorRepositoryMock)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo:        mockRepo,
		currentClient:      mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}

	updatedAt, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	yearGroup := 10
	existingVisitor := &entity.Visitor{
		Id:             1,
		ErpID:          123,
		UpdatedAt:      updatedAt,
		ErpYearGroupID: int32(yearGroup),
		ErpDivisions:   []int32{},
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
	updated, err := sync.SaveStudent(student)

	assert.NoError(t, err)
	assert.False(t, updated)
	mockRepo.AssertNotCalled(t, "AddVisitor")
}

func TestSaveStudent_IsUpToDate_UpdateNeeded(t *testing.T) {
	mockRepo := new(providertest.VisitorRepositoryMock)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo:        mockRepo,
		currentClient:      mockClient,
		Logger:             zap.NewNop().Sugar(),
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

	mockRepo.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.Id == 1 && v.ErpID == 123
	})).Return(nil)

	updated, err := sync.SaveStudent(student)

	assert.NoError(t, err)
	assert.True(t, updated)
	mockRepo.AssertExpectations(t)
}

func TestSaveStudent_NewVisitor(t *testing.T) {
	mockRepo := new(providertest.VisitorRepositoryMock)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo:        mockRepo,
		currentClient:      mockClient,
		Logger:             zap.NewNop().Sugar(),
		yearGroupDivisions: make(map[int32][]int32),
	}
	sync.currentVisitors = []*entity.Visitor{}

	yearGroup := 10
	student := isams.Student{
		ID:        123,
		YearGroup: &yearGroup,
	}

	divisionsResp := &isams.YearGroupsDivisionsResponse{Divisions: []isams.Division{}}
	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(divisionsResp, nil)

	mockRepo.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.Id == 0
	})).Return(nil)

	updated, err := sync.SaveStudent(student)

	assert.NoError(t, err)
	assert.True(t, updated)
	mockRepo.AssertExpectations(t)
}

func TestSaveStudent_InvalidTimeFormat(t *testing.T) {
	mockRepo := new(providertest.VisitorRepositoryMock)
	mockClient := new(MockERPClient)
	sync := &StudentSync{
		VisitorRepo:        mockRepo,
		currentClient:      mockClient,
		Logger:             zap.NewNop().Sugar(),
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

	mockRepo.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpID == 123 && v.UpdatedAt.IsZero() == false // Should default to Now() in IsUpToDate if zero
	})).Return(nil)

	_, err := sync.SaveStudent(student)
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
	mockRepo := new(providertest.VisitorRepositoryMock)
	mockClient := new(MockERPClient)

	sync := &StudentSync{
		VisitorRepo:        mockRepo,
		currentClient:      mockClient,
		Logger:             zap.NewNop().Sugar(),
		yearGroupDivisions: make(map[int32][]int32),
	}

	yearGroup := 10
	student := isams.Student{
		ID:        123,
		YearGroup: &yearGroup,
	}

	mockClient.On("GetYearGroupDivisions", int32(yearGroup)).Return(nil, errors.New("division error"))

	_, err := sync.SaveStudent(student)

	assert.Error(t, err)
	assert.Equal(t, "division error", err.Error())
}

func TestGetDivisionsByYearGroup_Cache(t *testing.T) {
	mockClient := new(MockERPClient)
	sync := &StudentSync{
		currentClient:      mockClient,
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
		currentClient:      mockClient,
		yearGroupDivisions: make(map[int32][]int32),
	}

	yearGroup := int32(10)
	mockClient.On("GetYearGroupDivisions", yearGroup).Return(nil, errors.New("api error"))

	divs, err := sync.getDivisionsByYearGroup(yearGroup)
	assert.Error(t, err)
	assert.Nil(t, divs)
	assert.Equal(t, "api error", err.Error())
}

func TestSaveStudent_SaveVisitor(t *testing.T) {
	yGD := make(map[int32][]int32)
	yGD[6] = []int32{2}

	loc := time.FixedZone("+0100", 1*60*60)

	curVisitors := []*entity.Visitor{}
	existingVisitor := &entity.Visitor{
		Id:             1,
		ErpID:          5225,
		Name:           "Alice",
		Surname:        "Smith",
		Grade:          6,
		ErpSchoolID:    "132014861202",
		UpdatedAt:      time.Date(2024, 1, 1, 12, 0, 0, 0, loc),
		ErpYearGroupID: 6,
		ErpDivisions:   []int32{2},
	}
	curVisitors = append(curVisitors, existingVisitor)

	exVisitor := &entity.Visitor{
		Id:             1,
		ErpID:          5225,
		Name:           "John",
		Surname:        "Doe",
		FullName:       "John Doe",
		Grade:          6,
		ErpSchoolID:    "132014861202",
		IsStudent:      true,
		UpdatedAt:      time.Date(2025, 10, 6, 11, 56, 2, 0, loc),
		ErpYearGroupID: 6,
		ErpDivisions:   []int32{2},
	}

	vRMock := new(providertest.VisitorRepositoryMock)
	vRMock.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		if v.ErpID != exVisitor.ErpID {
			return false
		}
		if !v.UpdatedAt.Equal(exVisitor.UpdatedAt) {
			return false
		}
		return v.FullName == exVisitor.FullName && v.IsStudent == exVisitor.IsStudent
	})).Once().Return(nil)

	sync := &StudentSync{
		VisitorRepo:        vRMock,
		yearGroupDivisions: yGD,
		currentVisitors:    curVisitors,
		Logger:             zap.NewNop().Sugar(),
	}

	str := string(testSingleStudentResponse)
	assert.NotEmpty(t, str)

	erpStudentResponse := isams.StudentsResponse{}

	err := json.Unmarshal(testSingleStudentResponse, &erpStudentResponse)
	assert.NoError(t, err)

	updated, err := sync.SaveStudent(erpStudentResponse.Students[0])
	assert.NoError(t, err)
	assert.True(t, updated)
	vRMock.AssertExpectations(t)
}

func TestSyncStudentPhotos_InitErrors(t *testing.T) {
	t.Run("start sync session error", func(t *testing.T) {
		mockFactory := new(MockERPFactory)
		logger, logBuffer := util.NewTestLogger()

		sync := &StudentSync{
			ERPFactory: mockFactory,
			Logger:     logger,
		}

		mockFactory.On("NewClient", mock.Anything).Once().Return(nil, errors.New("client error"))

		err := sync.SyncStudentPhotos()

		assert.Error(t, err)
		assert.Equal(t, "client error", err.Error())
		assert.Contains(t, logBuffer.String(), "sync students photos: create ERP client failed")
	})

	t.Run("get all visitors error", func(t *testing.T) {
		mockFactory := new(MockERPFactory)
		mockClient := new(MockERPClient)
		mockRepo := new(providertest.VisitorRepositoryMock)
		logger, logBuffer := util.NewTestLogger()

		sync := &StudentSync{
			ERPFactory:  mockFactory,
			VisitorRepo: mockRepo,
			Logger:      logger,
		}

		mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)
		mockRepo.On("GetAll").Return(nil, errors.New("db error"))

		err := sync.SyncStudentPhotos()

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Contains(t, logBuffer.String(), "sync students photos: get all visitors failed")
	})
}

func TestSyncStudentPhotos_NotStudentsWillNotSync(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	visitors := []*entity.Visitor{
		{ErpSchoolID: "S1", IsStudent: false},
		{ErpSchoolID: "S2", IsStudent: false},
	}
	mockRepo.On("GetAll").Return(visitors, nil)

	err := sync.SyncStudentPhotos()

	assert.NoError(t, err)
	mockClient.AssertNotCalled(t, "GetStudentPhoto", mock.Anything)
}

func TestSyncStudentPhotos_PhotoNotFoundSkips(t *testing.T) {
	oldOsWriteFile := osWriteFile
	defer func() { osWriteFile = oldOsWriteFile }()
	
	fileWritten := false

	osWriteFile = func(filename string, data []byte, perm os.FileMode) error {
		fileWritten = true
		return nil
	}

	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	visitors := []*entity.Visitor{
		{ErpSchoolID: "S1", IsStudent: true},
	}
	mockRepo.On("GetAll").Return(visitors, nil)

	mockClient.On("GetStudentPhoto", "S1").Once().Return(nil, isams.ErrStudentPhotoNotFound)

	err := sync.SyncStudentPhotos()

	assert.NoError(t, err)
	assert.False(t, fileWritten)
	mockClient.AssertExpectations(t)
}

func TestSyncStudentPhotos_OtherGetPhotoError(t *testing.T) {
	oldOsWriteFile := osWriteFile
	defer func() { osWriteFile = oldOsWriteFile }()
	
	fileWritten := false

	osWriteFile = func(filename string, data []byte, perm os.FileMode) error {
		fileWritten = true
		return nil
	}

	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)
	logger, logBuffer := util.NewTestLogger()

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      logger,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	visitors := []*entity.Visitor{
		{ErpSchoolID: "S1", IsStudent: true},
	}
	mockRepo.On("GetAll").Return(visitors, nil)

	mockClient.On("GetStudentPhoto", "S1").Once().Return(nil, isams.ErrAPIResponseBody)

	err := sync.SyncStudentPhotos()

	assert.NoError(t, err)
	assert.False(t, fileWritten)
	mockClient.AssertExpectations(t)
	logStr := logBuffer.String() 
	assert.Contains(t, logStr, "sync students photos: get student photo failed")
	assert.Contains(t, logStr, `"error": "invalid API response body"`)
}

func TestSyncStudentPhotos_ResponsePhotoIsEmpty(t *testing.T) {
	oldOsWriteFile := osWriteFile
	defer func() { osWriteFile = oldOsWriteFile }()
	
	fileWritten := false

	osWriteFile = func(filename string, data []byte, perm os.FileMode) error {
		fileWritten = true
		return nil
	}

	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	visitors := []*entity.Visitor{
		{ErpSchoolID: "S1", IsStudent: true},
	}
	mockRepo.On("GetAll").Return(visitors, nil)

	emptyDataResponse := &isams.StudentPhotoResponse{}
	mockClient.On("GetStudentPhoto", "S1").Once().Return(emptyDataResponse, nil)

	err := sync.SyncStudentPhotos()

	assert.NoError(t, err)
	assert.False(t, fileWritten)
	mockClient.AssertExpectations(t)
}

func TestSyncStudentPhotos_PhotoAddedSuccess(t *testing.T) {
	oldOsWriteFile := osWriteFile
	defer func() { osWriteFile = oldOsWriteFile }()
	
	fileWritten := false

	osWriteFile = func(filename string, data []byte, perm os.FileMode) error {
		assert.Equal(t, "storage-path/mypath/S1.png", filename)
		assert.Equal(t, " some photo data ", string(data))
		assert.Equal(t, os.FileMode(0644), perm)		
		fileWritten = true
		return nil
	}

	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)
	cfg := &config.Config{
		StudentsImagePhotoDir: "storage-path/mypath",
		StudentsImagePhotoURLPrefix: "url-prefix/mypath",
	}

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
		Config:      cfg,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	visitors := []*entity.Visitor{
		{ErpSchoolID: "S1", IsStudent: true},
	}
	mockRepo.On("GetAll").Return(visitors, nil)

	someDataStr := " some photo data "
	dataResponse := &isams.StudentPhotoResponse{
		Extension: "png",
		Data: []byte(someDataStr),
	}
	mockClient.On("GetStudentPhoto", "S1").Once().Return(dataResponse, nil)

	mockRepo.On("SaveVisitor", mock.MatchedBy(func(v *entity.Visitor) bool {
		return v.ErpSchoolID == "S1" && v.Image == "url-prefix/mypath/S1.png"
	})).Once().Return(nil)

	err := sync.SyncStudentPhotos()

	assert.NoError(t, err)
	assert.True(t, fileWritten)
	mockClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestSyncStudentPhotos_SaveImagePathError(t *testing.T) {
	oldOsWriteFile := osWriteFile
	defer func() { osWriteFile = oldOsWriteFile }()
	
	fileWritten := false

	osWriteFile = func(filename string, data []byte, perm os.FileMode) error {
		assert.Equal(t, "storage-path/mypath/S1.png", filename)
		assert.Equal(t, " some photo data ", string(data))
		assert.Equal(t, os.FileMode(0644), perm)		
		fileWritten = true
		return nil
	}

	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)
	cfg := &config.Config{
		StudentsImagePhotoDir: "storage-path/mypath",
		StudentsImagePhotoURLPrefix: "url-prefix/mypath",
	}

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
		Config:      cfg,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	visitors := []*entity.Visitor{
		{ErpSchoolID: "S1", IsStudent: true},
	}
	mockRepo.On("GetAll").Return(visitors, nil)

	someDataStr := " some photo data "
	dataResponse := &isams.StudentPhotoResponse{
		Extension: "png",
		Data: []byte(someDataStr),
	}
	mockClient.On("GetStudentPhoto", "S1").Once().Return(dataResponse, nil)

	mockRepo.On("SaveVisitor", mock.Anything).Return(assert.AnError)

	err := sync.SyncStudentPhotos()

	assert.Error(t, err)
	assert.True(t, fileWritten)
	mockClient.AssertExpectations(t)
}

func TestSyncStudentPhotos_WriteImageError(t *testing.T) {
	oldOsWriteFile := osWriteFile
	defer func() { osWriteFile = oldOsWriteFile }()
	
	fileWritten := false

	osWriteFile = func(filename string, data []byte, perm os.FileMode) error {
		assert.Equal(t, "storage-path/mypath/S1.png", filename)
		assert.Equal(t, " some photo data ", string(data))
		assert.Equal(t, os.FileMode(0644), perm)		
		fileWritten = true
		return fmt.Errorf("file write error")
	}

	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)
	cfg := &config.Config{
		StudentsImagePhotoDir: "storage-path/mypath",
		StudentsImagePhotoURLPrefix: "url-prefix/mypath",
	}

	sync := &StudentSync{
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      zap.NewNop().Sugar(),
		Config:      cfg,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)

	visitors := []*entity.Visitor{
		{ErpSchoolID: "S1", IsStudent: true},
	}
	mockRepo.On("GetAll").Return(visitors, nil)

	someDataStr := " some photo data "
	dataResponse := &isams.StudentPhotoResponse{
		Extension: "png",
		Data: []byte(someDataStr),
	}
	mockClient.On("GetStudentPhoto", "S1").Once().Return(dataResponse, nil)

	err := sync.SyncStudentPhotos()

	assert.Error(t, err)
	assert.Equal(t, "file write error", err.Error())
	assert.True(t, fileWritten)
	mockRepo.AssertNotCalled(t, "SaveVisitor")
	mockClient.AssertExpectations(t)
}

func TestOsWriteFile_Default(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sample.txt"

	err := osWriteFile(path, []byte("data"), 0644)

	assert.NoError(t, err)
	contents, readErr := os.ReadFile(path)
	assert.NoError(t, readErr)
	assert.Equal(t, "data", string(contents))
}

func TestSyncRegistrationCodesDictionaries_StartSyncSessionError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	logger, logBuffer := util.NewTestLogger()
	sync := &StudentSync{
		ERPFactory: mockFactory,
		Logger:     logger,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(nil, errors.New("client error"))

	err := sync.SyncRegistrationCodesDictionaries()

	assert.Error(t, err)
	assert.Equal(t, "client error", err.Error())
	assert.Contains(t, logBuffer.String(), "sync registration codes: create ERP client failed")
	mockFactory.AssertExpectations(t)
}

func TestSyncRegistrationCodesDictionaries_AbsenceCodesError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	logger, logBuffer := util.NewTestLogger()

	oldAbsence := entity.GetAbsenceCodeDictionary()
	oldPresent := entity.GetPresentsCodeDictionary()
	sentinelAbsence := &entity.RegistrationCodeDictionary{
		Codes: map[int32]*entity.RegistrationCode{
			1: {ID: 1, Code: "ABS", Name: "Absence", IsAbsenceCode: true},
		},
		UploadedAt: time.Now().Add(-time.Hour),
	}
	entity.SetAbsenceCodeDictionary(sentinelAbsence)
	t.Cleanup(func() {
		entity.SetAbsenceCodeDictionary(oldAbsence)
		entity.SetPresentsCodeDictionary(oldPresent)
	})

	sync := &StudentSync{
		ERPFactory: mockFactory,
		Logger:     logger,
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)
	mockClient.On("GetRegistrationAbsenceCodes").Once().Return(nil, errors.New("absence error"))

	err := sync.SyncRegistrationCodesDictionaries()

	assert.Error(t, err)
	assert.Equal(t, "absence error", err.Error())
	assert.Contains(t, logBuffer.String(), "sync registration codes: get absence codes failed")
	assert.Same(t, sentinelAbsence, entity.GetAbsenceCodeDictionary())
	assert.Same(t, oldPresent, entity.GetPresentsCodeDictionary())
	mockFactory.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestSyncRegistrationCodesDictionaries_PresentCodesError(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	logger, logBuffer := util.NewTestLogger()

	oldAbsence := entity.GetAbsenceCodeDictionary()
	oldPresent := entity.GetPresentsCodeDictionary()
	sentinelPresent := &entity.RegistrationCodeDictionary{
		Codes: map[int32]*entity.RegistrationCode{
			2: {ID: 2, Code: "P", Name: "Present", IsAbsenceCode: false},
		},
		UploadedAt: time.Now().Add(-time.Hour),
	}
	entity.SetPresentsCodeDictionary(sentinelPresent)
	t.Cleanup(func() {
		entity.SetAbsenceCodeDictionary(oldAbsence)
		entity.SetPresentsCodeDictionary(oldPresent)
	})

	sync := &StudentSync{
		ERPFactory: mockFactory,
		Logger:     logger,
	}

	absenceResp := &isams.RegistrationAbsenceCodesResponse{
		AbsenceCodes: []isams.RegistrationAbsenceCode{
			{ID: 10, Code: "A", Name: "Absence Code"},
		},
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)
	mockClient.On("GetRegistrationAbsenceCodes").Once().Return(absenceResp, nil)
	mockClient.On("GetRegistrationPresentCodes").Once().Return(nil, errors.New("present error"))

	err := sync.SyncRegistrationCodesDictionaries()

	assert.Error(t, err)
	assert.Equal(t, "present error", err.Error())
	assert.Contains(t, logBuffer.String(), "sync registration codes: get present codes failed")
	absenceDict := entity.GetAbsenceCodeDictionary()
	if assert.NotNil(t, absenceDict) {
		assert.Contains(t, absenceDict.Codes, int32(10))
		assert.True(t, absenceDict.Codes[10].IsAbsenceCode)
	}
	assert.Same(t, sentinelPresent, entity.GetPresentsCodeDictionary())
	mockFactory.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestSyncRegistrationCodesDictionaries_Success(t *testing.T) {
	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)

	oldAbsence := entity.GetAbsenceCodeDictionary()
	oldPresent := entity.GetPresentsCodeDictionary()
	t.Cleanup(func() {
		entity.SetAbsenceCodeDictionary(oldAbsence)
		entity.SetPresentsCodeDictionary(oldPresent)
	})

	sync := &StudentSync{
		ERPFactory: mockFactory,
	}

	absenceResp := &isams.RegistrationAbsenceCodesResponse{
		AbsenceCodes: []isams.RegistrationAbsenceCode{
			{ID: 10, Code: "A", Name: "Absence Code"},
		},
	}
	presentResp := &isams.RegistrationPresentCodeResponse{
		PresentCodes: []isams.RegistrationPresentCode{
			{ID: 20, Code: "P", Name: "Present Code"},
		},
	}

	mockFactory.On("NewClient", mock.Anything).Once().Return(mockClient, nil)
	mockClient.On("GetRegistrationAbsenceCodes").Once().Return(absenceResp, nil)
	mockClient.On("GetRegistrationPresentCodes").Once().Return(presentResp, nil)

	err := sync.SyncRegistrationCodesDictionaries()

	assert.NoError(t, err)
	absenceDict := entity.GetAbsenceCodeDictionary()
	if assert.NotNil(t, absenceDict) {
		assert.False(t, absenceDict.UploadedAt.IsZero())
		assert.Contains(t, absenceDict.Codes, int32(10))
		assert.True(t, absenceDict.Codes[10].IsAbsenceCode)
	}
	presentDict := entity.GetPresentsCodeDictionary()
	if assert.NotNil(t, presentDict) {
		assert.False(t, presentDict.UploadedAt.IsZero())
		assert.Contains(t, presentDict.Codes, int32(20))
		assert.False(t, presentDict.Codes[20].IsAbsenceCode)
	}
	mockFactory.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}
