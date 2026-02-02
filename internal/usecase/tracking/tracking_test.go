package tracking

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

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

type trackingTestEnv struct {
	tracker     *StudentTracker
	mockFactory *MockERPFactory
	mockClient  *MockERPClient
	mockRepo    *providertest.VisitorRepositoryMock
	logs        *observer.ObservedLogs
}

func prepareTrackingTestEnv(t *testing.T) *trackingTestEnv {
	t.Helper()
	resetTrackingEnv(t)

	oldPresent := entity.GetPresentsCodeDictionary()
	oldAbsence := entity.GetAbsenceCodeDictionary()
	entity.SetPresentsCodeDictionary(&entity.RegistrationCodeDictionary{
		Codes: map[int32]*entity.RegistrationCode{
			10: {ID: 10, Code: "/", Name: "Present"},
		},
		UploadedAt: time.Now(),
	})
	entity.SetAbsenceCodeDictionary(&entity.RegistrationCodeDictionary{
		Codes: map[int32]*entity.RegistrationCode{
			11: {ID: 11, Code: "O", Name: "Absent", IsAbsenceCode: true},
		},
		UploadedAt: time.Now(),
	})
	t.Cleanup(func() {
		entity.SetPresentsCodeDictionary(oldPresent)
		entity.SetAbsenceCodeDictionary(oldAbsence)
	})

	cfg, err := config.NewFromEnv()
	assert.NoError(t, err)
	globalCfg := registerGlobalConfig(t, &cfg)

	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core).Sugar()

	mockFactory := new(MockERPFactory)
	mockClient := new(MockERPClient)
	mockRepo := new(providertest.VisitorRepositoryMock)

	util.SetAppLocal(time.UTC)
	t.Cleanup(func() {
		util.SetAppLocal(time.Local)
	})

	tracker := &StudentTracker{
		Cfg:         globalCfg,
		ERPFactory:  mockFactory,
		VisitorRepo: mockRepo,
		Logger:      logger,
	}

	return &trackingTestEnv{
		tracker:     tracker,
		mockFactory: mockFactory,
		mockClient:  mockClient,
		mockRepo:    mockRepo,
		logs:        logs,
	}
}

func registerGlobalConfig(t *testing.T, cfg *config.Config) *config.Config {
	t.Helper()
	if err := container.Singleton(func() *config.Config { return cfg }); err != nil {
		var existing *config.Config
		container.MustResolve(container.Global, &existing)
		*existing = *cfg
		return existing
	}
	return cfg
}

func resetTrackingEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"ERP_LOCAL_TIMEZONE",
		"APP_LOCAL_TIMEZONE",
		"ERP_MAIN_REGISTRATION_PERIOD_TYPE",
		"ERP_DEFAULT_PRESENT_CODE_NAME",
		"ERP_DEFAULT_LESSON_ABSENCE_CODE_NAME",
	}

	previous := map[string]*string{}
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			val := value
			previous[key] = &val
		} else {
			previous[key] = nil
		}
		_ = os.Unsetenv(key)
	}

	_ = os.Setenv("ERP_LOCAL_TIMEZONE", "UTC")
	_ = os.Setenv("APP_LOCAL_TIMEZONE", "UTC")
	_ = os.Setenv("ERP_MAIN_REGISTRATION_PERIOD_TYPE", "AM")
	_ = os.Setenv("ERP_DEFAULT_PRESENT_CODE_NAME", "/")
	_ = os.Setenv("ERP_DEFAULT_LESSON_ABSENCE_CODE_NAME", "O")

	t.Cleanup(func() {
		for key, value := range previous {
			if value == nil {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, *value)
		}
	})
}

func makePeriod(id int32, name, periodType string, start, at, finish time.Time) isams.RegistrationPeriod {
	return isams.RegistrationPeriod{
		ID:               id,
		FriendlyName:     name,
		RegistrationType: periodType,
		Start:            start.Format(time.RFC3339),
		Time:             at.Format(time.RFC3339),
		Finish:           finish.Format(time.RFC3339),
	}
}

func makeVisitor() *entity.Visitor {
	return &entity.Visitor{
		Id:           1,
		ErpSchoolID:  "S1",
		ErpDivisions: []int32{1},
	}
}

func TestStudentFilterConfigYearGroupSlice(t *testing.T) {
	val1 := int32(10)
	val2 := int32(11)
	cfg := &StudentFilterConfig{
		YearGroup: []*int32{&val1, nil, &val2},
	}
	assert.Equal(t, []int32{10, 11}, cfg.YearGroupSlice())

	cfg = &StudentFilterConfig{}
	assert.Empty(t, cfg.YearGroupSlice())
}

func TestWithYearGroups(t *testing.T) {
	cfg := &StudentFilterConfig{}
	WithYearGroups([]int32{1, 2})(cfg)
	assert.Len(t, cfg.YearGroup, 2)
	assert.Equal(t, int32(1), *cfg.YearGroup[0])
	assert.Equal(t, int32(2), *cfg.YearGroup[1])
}

func TestPrepareRegistrationStatusRequest(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	leave := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	comment := "note"
	item := &entity.AttendanceItem{
		IsPresent:             true,
		IsLate:                true,
		PresentCodeID:         ptrInt32(10),
		AbsenceCodeID:         ptrInt32(11),
		LeavingOrLeftDateTime: &leave,
		NumberOfMinutesLate:   5,
		RegistrationComment:   &comment,
	}
	req := env.tracker.PrepareRegistrationStatusRequest(item)
	assert.True(t, req.IsPresent)
	assert.True(t, req.IsLate)
	assert.Equal(t, ptrInt32(10), req.PresentCodeID)
	assert.Equal(t, ptrInt32(11), req.AbsenceCodeID)
	assert.Equal(t, int32(5), req.NumberOfMinutesLate)
	assert.Equal(t, &comment, req.RegistrationComment)
	assert.NotNil(t, req.LeavingOrLeftDateTime)
}

func TestPrepareRegistrationStatusRequestWithoutLeavingTime(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	item := &entity.AttendanceItem{}
	req := env.tracker.PrepareRegistrationStatusRequest(item)
	assert.Nil(t, req.LeavingOrLeftDateTime)
}

func TestGetPeriodsLogsAndContinues(t *testing.T) {
	env := prepareTrackingTestEnv(t)

	resp := &isams.RegistrationPeriodsResponse{
		RegistrationPeriods: []isams.RegistrationPeriod{
			makePeriod(100, "AM", "AM", time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC), time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC), time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)),
		},
	}

	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(nil, errors.New("division error"))
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(2)).Return(resp, nil)

	schedule, err := env.tracker.getPeriods(env.mockClient, []int32{1, 2})
	assert.NoError(t, err)
	assert.Len(t, schedule.Periods, 1)
	assert.True(t, env.logs.FilterMessageSnippet("Error getting registration periods for division 1").Len() > 0)

	env.mockClient.AssertExpectations(t)
}

func TestAddPeriodsFromResponseSkipsInvalidTimes(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	schedule := &entity.Schedule{}

	resp := &isams.RegistrationPeriodsResponse{
		RegistrationPeriods: []isams.RegistrationPeriod{
			{ID: 1, FriendlyName: "BadStart", RegistrationType: "AM", Start: "bad", Time: time.Now().Format(time.RFC3339), Finish: time.Now().Add(time.Hour).Format(time.RFC3339)},
			{ID: 2, FriendlyName: "BadFinish", RegistrationType: "AM", Start: time.Now().Format(time.RFC3339), Time: time.Now().Format(time.RFC3339), Finish: "bad"},
			{ID: 3, FriendlyName: "BadTime", RegistrationType: "AM", Start: time.Now().Format(time.RFC3339), Time: "bad", Finish: time.Now().Add(time.Hour).Format(time.RFC3339)},
			makePeriod(4, "Good", "AM", time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC), time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC), time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)),
		},
	}

	env.tracker.addPeriodsFromResponse(schedule, resp)
	assert.Len(t, schedule.Periods, 1)
	_, ok := schedule.Periods[4]
	assert.True(t, ok)
}

func TestFillAttendanceInfo(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()
	schedule := &entity.Schedule{Periods: map[int32]*entity.RegistrationPeriod{
		100: {ID: 100, Name: "AM", Type: "AM"},
		101: {ID: 101, Name: "P1", Type: "LESSON"},
		102: {ID: 102, Name: "P2", Type: "LESSON"},
		103: {ID: 103, Name: "P3", Type: "LESSON"},
	}}
	attendance := entity.NewStudentAttendance(student, schedule)

	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(nil, errors.New("status error"))

	invalidLeave := "bad"
	status1 := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 101, LeavingOrLeftDateTime: &invalidLeave, IsRegistered: 0}
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(101)).Return(status1, nil)

	emptyLeave := ""
	status2 := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 102, LeavingOrLeftDateTime: &emptyLeave, IsRegistered: 1, IsPresent: true}
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(102)).Return(status2, nil)

	leave := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC).Format(time.RFC3339)
	status3 := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 103, LeavingOrLeftDateTime: &leave, IsRegistered: 1, IsPresent: true}
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(103)).Return(status3, nil)

	env.tracker.fillAttendanceInfo(env.mockClient, attendance)

	env.mockClient.AssertExpectations(t)
}

func TestTrack_NewClientErrorLogs(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	env.mockFactory.On("NewClient", mock.Anything).Return(nil, errors.New("factory error"))

	err := env.tracker.Track(context.Background(), makeVisitor())
	assert.Error(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("Error creating ERP client for student 1").Len() > 0)

	env.mockFactory.AssertExpectations(t)
}

func TestTrack_NoUpdate(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	period := makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish)

	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{period}}
	status := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 1, IsPresent: true}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(status, nil)

	err := env.tracker.Track(context.Background(), student)
	assert.NoError(t, err)

	env.mockClient.AssertExpectations(t)
	env.mockFactory.AssertExpectations(t)
}

func TestTrack_GetPeriodsErrorLogs(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	env.mockFactory.On("NewClient", mock.Anything).Return(erp.Client(nil), nil)

	err := env.tracker.Track(context.Background(), student)
	assert.Error(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("Error getting periods for student 1").Len() > 0)
}

func TestTrack_UpdateMainAndPeriods(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	period1Start := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Time := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Finish := time.Date(2026, 2, 1, 8, 59, 0, 0, time.UTC)

	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
		makePeriod(101, "P1", "LESSON", period1Start, period1Time, period1Finish),
	}}

	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}
	statusP1 := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 101, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(101)).Return(statusP1, nil)

	env.mockClient.On("PutRegistration", "S1", int32(100), mock.Anything).Return(nil)
	env.mockClient.On("PutRegistration", "S1", int32(101), mock.Anything).Return(nil)

	util.SetCurrentTime(8, 30)
	err := env.tracker.Track(context.Background(), student)
	assert.NoError(t, err)

	env.mockClient.AssertExpectations(t)
	env.mockFactory.AssertExpectations(t)
}

func TestTrack_PutRegistrationErrorInPeriodLoopLogs(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	period1Start := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Time := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Finish := time.Date(2026, 2, 1, 8, 59, 0, 0, time.UTC)

	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
		makePeriod(101, "P1", "LESSON", period1Start, period1Time, period1Finish),
	}}

	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}
	statusP1 := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 101, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(101)).Return(statusP1, nil)

	env.mockClient.On("PutRegistration", "S1", int32(100), mock.Anything).Return(nil)
	env.mockClient.On("PutRegistration", "S1", int32(101), mock.Anything).Return(errors.New("put error"))

	util.SetCurrentTime(8, 30)
	err := env.tracker.Track(context.Background(), student)
	assert.Error(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("Error updating registration for student 1, period 101").Len() > 0)
}

func TestTrack_MainUpdateOnly(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	period1Start := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Time := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Finish := time.Date(2026, 2, 1, 8, 59, 0, 0, time.UTC)

	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
		makePeriod(101, "P1", "LESSON", period1Start, period1Time, period1Finish),
	}}

	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}
	statusP1 := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 101, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(101)).Return(statusP1, nil)

	env.mockClient.On("PutRegistration", "S1", int32(100), mock.Anything).Return(nil)

	util.SetCurrentTime(7, 50)
	err := env.tracker.Track(context.Background(), student)
	assert.NoError(t, err)

	env.mockClient.AssertExpectations(t)
	env.mockFactory.AssertExpectations(t)
}

func TestTrack_PutRegistrationErrorLogs(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
	}}
	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.On("PutRegistration", "S1", int32(100), mock.Anything).Return(errors.New("put error"))

	util.SetCurrentTime(7, 50)
	err := env.tracker.Track(context.Background(), student)
	assert.Error(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("Error updating registration for student 1, period 100").Len() > 0)
}

func TestTrack_TrackForbyPeriodsError(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	oldAbsence := entity.GetAbsenceCodeDictionary()
	entity.SetAbsenceCodeDictionary(nil)
	t.Cleanup(func() {
		entity.SetAbsenceCodeDictionary(oldAbsence)
	})

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	period1Start := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Time := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	period1Finish := time.Date(2026, 2, 1, 8, 59, 0, 0, time.UTC)

	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
		makePeriod(101, "P1", "LESSON", period1Start, period1Time, period1Finish),
	}}

	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}
	statusP1 := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 101, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(101)).Return(statusP1, nil)
	env.mockClient.On("PutRegistration", "S1", int32(100), mock.Anything).Return(nil)

	util.SetCurrentTime(7, 50)
	err := env.tracker.Track(context.Background(), student)
	assert.Error(t, err)
}

func TestTrack_MainPeriodNotFound(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()
	env.tracker.Cfg.ERPMainRegistrationPeriodType = "TT"

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
	}}
	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)

	util.SetCurrentTime(7, 50)
	err := env.tracker.Track(context.Background(), student)
	assert.Error(t, err)
}

func TestTrackUntrackedStudentsAsAbsence_FindAllError(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	env.mockRepo.On("FindAll", mock.Anything).Return(nil, errors.New("repo error"))

	err := env.tracker.TrackUntrackedStudentsAsAbsence(context.Background())
	assert.Error(t, err)
	env.mockRepo.AssertExpectations(t)
}

func TestTrackUntrackedStudentsAsAbsence_LogsErrors(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	students := []*entity.Visitor{
		{Id: 1, ErpSchoolID: "S1", ErpDivisions: []int32{1}},
		{Id: 2, ErpSchoolID: "S2", ErpDivisions: []int32{1}},
	}
	env.mockRepo.On("FindAll", mock.Anything).Return(students, nil)
	env.mockFactory.On("NewClient", mock.Anything).Return(nil, errors.New("factory error"))

	err := env.tracker.TrackUntrackedStudentsAsAbsence(context.Background())
	assert.NoError(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("Error tracking absence for student 1").Len() > 0)
	assert.True(t, env.logs.FilterMessageSnippet("Error tracking absence for student 2").Len() > 0)
}

func TestTrackUntrackedStudentsAsAbsence_WithYearGroups(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	env.mockRepo.On("FindAll", mock.Anything).Return([]*entity.Visitor{}, nil)

	err := env.tracker.TrackUntrackedStudentsAsAbsence(context.Background(), WithYearGroups([]int32{1, 2}))
	assert.NoError(t, err)
	env.mockRepo.AssertExpectations(t)
}

func TestTrackAbsenceForNotRegisteredStudent_Success(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
	}}
	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.On("PutRegistration", "S1", int32(100), mock.Anything).Return(nil)

	util.SetCurrentTime(8, 0)
	err := env.tracker.trackAbsenceForNotRegisteredStudent(context.Background(), student)
	assert.NoError(t, err)
	env.mockClient.AssertExpectations(t)
}

func TestTrackAbsenceForNotRegisteredStudent_GetPeriodsErrorLogs(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	env.mockFactory.On("NewClient", mock.Anything).Return(erp.Client(nil), nil)

	err := env.tracker.trackAbsenceForNotRegisteredStudent(context.Background(), student)
	assert.Error(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("Error getting periods for student 1").Len() > 0)
}

func TestTrackAbsenceForNotRegisteredStudent_NoUpdate(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
	}}
	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.AssertNotCalled(t, "PutRegistration")

	util.SetCurrentTime(7, 50)
	err := env.tracker.trackAbsenceForNotRegisteredStudent(context.Background(), student)
	assert.NoError(t, err)
	env.mockClient.AssertExpectations(t)
}

func TestTrackAbsenceForNotRegisteredStudent_MainPeriodNotFound(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()
	env.tracker.Cfg.ERPMainRegistrationPeriodType = "TT"

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
	}}
	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)

	util.SetCurrentTime(8, 0)
	err := env.tracker.trackAbsenceForNotRegisteredStudent(context.Background(), student)
	assert.NoError(t, err)
	env.mockClient.AssertNotCalled(t, "PutRegistration", mock.Anything, mock.Anything, mock.Anything)
}

func TestTrackAbsenceForNotRegisteredStudent_DefaultAbsenceCodeMissing(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	oldAbsence := entity.GetAbsenceCodeDictionary()
	entity.SetAbsenceCodeDictionary(nil)
	t.Cleanup(func() {
		entity.SetAbsenceCodeDictionary(oldAbsence)
	})

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
	}}
	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)

	util.SetCurrentTime(8, 0)
	err := env.tracker.trackAbsenceForNotRegisteredStudent(context.Background(), student)
	assert.Error(t, err)
}

func TestTrackAbsenceForNotRegisteredStudent_PutErrorLogs(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()

	mainStart := time.Date(2026, 2, 1, 7, 37, 0, 0, time.UTC)
	mainTime := time.Date(2026, 2, 1, 7, 40, 0, 0, time.UTC)
	mainFinish := time.Date(2026, 2, 1, 7, 55, 0, 0, time.UTC)
	resp := &isams.RegistrationPeriodsResponse{RegistrationPeriods: []isams.RegistrationPeriod{
		makePeriod(100, "AM", "AM", mainStart, mainTime, mainFinish),
	}}
	statusMain := &isams.RegistrationStatus{SchoolID: "S1", RegistrationPeriodID: 100, IsRegistered: 0, IsPresent: false}

	env.mockFactory.On("NewClient", mock.Anything).Return(env.mockClient, nil)
	env.mockClient.On("GetCurrentRegistrationPeriodsForDivision", int32(1)).Return(resp, nil)
	env.mockClient.On("GetRegistrationStatusForStudent", "S1", int32(100)).Return(statusMain, nil)
	env.mockClient.On("PutRegistration", "S1", int32(100), mock.Anything).Return(errors.New("put error"))

	util.SetCurrentTime(8, 0)
	err := env.tracker.trackAbsenceForNotRegisteredStudent(context.Background(), student)
	assert.Error(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("AutoAbsence: Error updating registration for student 1, period 100").Len() > 0)
}

func TestTrackAbsenceForNotRegisteredStudent_NewClientErrorLogs(t *testing.T) {
	env := prepareTrackingTestEnv(t)
	student := makeVisitor()
	env.mockFactory.On("NewClient", mock.Anything).Return(nil, errors.New("factory error"))

	err := env.tracker.trackAbsenceForNotRegisteredStudent(context.Background(), student)
	assert.Error(t, err)
	assert.True(t, env.logs.FilterMessageSnippet("Error creating ERP client for student 1").Len() > 0)
}

func ptrInt32(val int32) *int32 {
	return &val
}
