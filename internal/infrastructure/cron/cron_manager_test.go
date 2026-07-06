package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/go-co-op/gocron/v2"
	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type stubERPClient struct{}

func (s *stubERPClient) GetStudents(_, _ int32) (*isams.StudentsResponse, error) {
	return &isams.StudentsResponse{}, nil
}

func (s *stubERPClient) GetYearGroupDivisions(_ int32) (*isams.YearGroupsDivisionsResponse, error) {
	return &isams.YearGroupsDivisionsResponse{}, nil
}

func (s *stubERPClient) GetCurrentRegistrationPeriodsForDivision(_ int32) (*isams.RegistrationPeriodsResponse, error) {
	return &isams.RegistrationPeriodsResponse{}, nil
}

func (s *stubERPClient) GetRegistrationStatusForStudent(_ string, _ int32) (*isams.RegistrationStatus, error) {
	return &isams.RegistrationStatus{}, nil
}

func (s *stubERPClient) GetRegistrationAbsenceCodes() (*isams.RegistrationAbsenceCodesResponse, error) {
	return &isams.RegistrationAbsenceCodesResponse{}, nil
}

func (s *stubERPClient) GetRegistrationPresentCodes() (*isams.RegistrationPresentCodeResponse, error) {
	return &isams.RegistrationPresentCodeResponse{}, nil
}

func (s *stubERPClient) PutRegistration(_ string, _ int32, _ isams.RegistrationStatusRequest) error {
	return nil
}

func (s *stubERPClient) GetStudentPhoto(_ string) (*isams.StudentPhotoResponse, error) {
	return &isams.StudentPhotoResponse{}, nil
}

type stubERPFactory struct{}

func (s *stubERPFactory) NewClient(_ context.Context) (erp.Client, error) {
	return &stubERPClient{}, nil
}

type visitorRepoSpy struct {
	optsLen    int
	yearGroups []int32
}

func (v *visitorRepoSpy) GetAll() ([]*entity.Visitor, error) {
	return nil, nil
}

func (v *visitorRepoSpy) FindAll(opts ...provider.VisitorFilterOption) ([]*entity.Visitor, error) {
	v.optsLen = len(opts)
	cfg := &provider.VisitorFilterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	v.yearGroups = extractYearGroups(cfg.ERPYearGroup)
	return []*entity.Visitor{}, nil
}

func (v *visitorRepoSpy) FindById(_ int32) (*entity.Visitor, error) {
	return nil, nil
}

func (v *visitorRepoSpy) FindByKey(_ string) (*entity.VisitDetails, error) {
	return nil, nil
}

func (v *visitorRepoSpy) AddKeyToVisitor(_ *entity.Visitor, _ string) error {
	return nil
}

func (v *visitorRepoSpy) AddVisitor(_ *entity.Visitor) error {
	return nil
}

func (v *visitorRepoSpy) SaveVisitor(_ *entity.Visitor) error {
	return nil
}

func (v *visitorRepoSpy) DeleteVisitor(_ int32) error {
	return nil
}

func (v *visitorRepoSpy) RemoveKeyFromVisitor(_ int32, _ string) error {
	return nil
}

func (v *visitorRepoSpy) FindKeysByVisitorId(_ int32) ([]string, error) {
	return nil, nil
}

func (v *visitorRepoSpy) UpdateVisitorImage(_ int32, _ string) error {
	return nil
}

func extractYearGroups(values []*int32) []int32 {
	if values == nil {
		return nil
	}
	result := make([]int32, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}

func TestMarkNotRegisteredStudentsAsAbsent_UsesConfiguredYearGroups(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() {
		container.Global = oldGlobal
	})

	cfg := &config.Config{
		AutoRegistrationYearGroups: []int32{6, 7},
	}
	repo := &visitorRepoSpy{}

	container.MustSingleton(container.Global, func() *config.Config { return cfg })
	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitorRepository { return repo })
	container.MustSingleton(container.Global, func() erp.Factory { return &stubERPFactory{} })

	MarkNotRegisteredStudentsAsAbsent()()

	assert.Equal(t, 1, repo.optsLen)
	assert.Equal(t, []int32{6, 7}, repo.yearGroups)
}

func TestMarkNotRegisteredStudentsAsAbsent_WithoutConfiguredYearGroups(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() {
		container.Global = oldGlobal
	})

	cfg := &config.Config{
		AutoRegistrationYearGroups: nil,
	}
	repo := &visitorRepoSpy{}

	container.MustSingleton(container.Global, func() *config.Config { return cfg })
	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitorRepository { return repo })
	container.MustSingleton(container.Global, func() erp.Factory { return &stubERPFactory{} })

	MarkNotRegisteredStudentsAsAbsent()()

	assert.Equal(t, 1, repo.optsLen)
	assert.Empty(t, repo.yearGroups)
}

func TestRunCron_WhenERPNotIntegrated_ReturnsNoopAndNoError(t *testing.T) {
	cfg := &config.Config{
		ERPIntegrationEnabled: false,
		ISAMSBaseURL:          "",
		ISAMSAPIClientID:      "",
		ISAMSAPIClientSecret:  "",
		CronFinalizeReports:   "30 0 * * *",
	}

	stopFunc, err := RunCron(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, stopFunc)
	// Calling stop should not panic
	stopFunc()
}

func TestRunCron_WhenERPIntegrated_ReturnsStopFuncAndNoError(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() {
		container.Global = oldGlobal
	})

	// Setup minimal container dependencies for jobs that run immediately
	container.MustSingleton(container.Global, func() *config.Config {
		return &config.Config{}
	})
	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitorRepository { return &visitorRepoSpy{} })
	container.MustSingleton(container.Global, func() erp.Factory { return &stubERPFactory{} })

	cfg := &config.Config{
		ERPIntegrationEnabled:     true,
		ISAMSBaseURL:              "https://example.com",
		ISAMSAPIClientID:          "client-id",
		ISAMSAPIClientSecret:      "client-secret",
		ForceERPSyncOnStart:       false,
		CronStudentSync:           "0 7-17/2 * * 1-5",
		CronPhotoSync:             "0 5 * * 1-5",
		CronRegistrationCodesSync: "0 7-17/1 * * 1-5",
		CronMarkAbsent:            "10 8-12/1 * * 1-5",
		CronFinalizeReports:       "30 0 * * *",
	}

	stopFunc, err := RunCron(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, stopFunc)
	stopFunc()
}

func TestRunCron_WithCustomCronSchedules(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() {
		container.Global = oldGlobal
	})

	container.MustSingleton(container.Global, func() *config.Config {
		return &config.Config{}
	})
	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitorRepository { return &visitorRepoSpy{} })
	container.MustSingleton(container.Global, func() erp.Factory { return &stubERPFactory{} })

	cfg := &config.Config{
		ERPIntegrationEnabled:     true,
		ISAMSBaseURL:              "https://example.com",
		ISAMSAPIClientID:          "client-id",
		ISAMSAPIClientSecret:      "client-secret",
		ForceERPSyncOnStart:       false,
		CronStudentSync:           "*/10 * * * *",
		CronPhotoSync:             "0 3 * * *",
		CronRegistrationCodesSync: "*/30 * * * 1-5",
		CronMarkAbsent:            "15 9 * * 1-5",
		CronFinalizeReports:       "30 0 * * *",
	}

	stopFunc, err := RunCron(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, stopFunc)
	stopFunc()
}

func TestRegisterJobs_WithDefaultSchedules(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() {
		container.Global = oldGlobal
	})

	container.MustSingleton(container.Global, func() *config.Config {
		return &config.Config{}
	})
	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitorRepository { return &visitorRepoSpy{} })
	container.MustSingleton(container.Global, func() erp.Factory { return &stubERPFactory{} })

	s, err := gocron.NewScheduler()
	assert.NoError(t, err)
	defer func() { _ = s.Shutdown() }()

	cfg := &config.Config{
		CronStudentSync:           "0 7-17/2 * * 1-5",
		CronPhotoSync:             "0 5 * * 1-5",
		CronRegistrationCodesSync: "0 7-17/1 * * 1-5",
		CronMarkAbsent:            "10 8-12/1 * * 1-5",
	}

	err = registerERPJobs(s, cfg)
	assert.NoError(t, err)
}

func TestRegisterJobs_WithInvalidCronExpression(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() {
		container.Global = oldGlobal
	})

	container.MustSingleton(container.Global, func() *config.Config {
		return &config.Config{}
	})
	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitorRepository { return &visitorRepoSpy{} })
	container.MustSingleton(container.Global, func() erp.Factory { return &stubERPFactory{} })

	s, err := gocron.NewScheduler()
	assert.NoError(t, err)
	defer func() { _ = s.Shutdown() }()

	cfg := &config.Config{
		CronStudentSync: "invalid cron expression",
	}

	err = registerERPJobs(s, cfg)
	assert.Error(t, err)
}

func TestRegisterReportJobs_RegistersFinalizationJob(t *testing.T) {
	s, err := gocron.NewScheduler()
	assert.NoError(t, err)
	defer func() { _ = s.Shutdown() }()

	err = registerJobs(s, &config.Config{CronFinalizeReports: "30 0 * * *"})
	assert.NoError(t, err)
	assert.Len(t, s.Jobs(), 1)
}

func TestRegisterReportJobs_InvalidCronExpression(t *testing.T) {
	s, err := gocron.NewScheduler()
	assert.NoError(t, err)
	defer func() { _ = s.Shutdown() }()

	err = registerJobs(s, &config.Config{CronFinalizeReports: "not a cron"})
	assert.Error(t, err)
}

func TestRunCron_WhenERPNotIntegrated_StillRegistersReportJob(t *testing.T) {
	cfg := &config.Config{
		ERPIntegrationEnabled: false,
		CronFinalizeReports:   "30 0 * * *",
	}

	stopFunc, err := RunCron(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, stopFunc)
	stopFunc()
}

func TestRunCron_WhenReportCronInvalid_ReturnsError(t *testing.T) {
	cfg := &config.Config{
		ERPIntegrationEnabled: false,
		CronFinalizeReports:   "totally invalid",
	}

	stopFunc, err := RunCron(cfg)
	assert.Error(t, err)
	assert.Nil(t, stopFunc)
}

func TestFinalizeReportsFunc_FinalizesUnfinalizedDays(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() { container.Global = oldGlobal })

	repo := new(providertest.VisitDailyReportRepositoryMock)
	d1 := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	repo.On("UnfinalizedDaysBefore", mock.Anything).Return([]time.Time{d1, d2}, nil)
	repo.On("FinalizeDay", d1).Return(nil)
	repo.On("FinalizeDay", d2).Return(errors.New("finalize failed")) // error is logged, not fatal

	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitDailyReportRepository { return repo })

	FinalizeReportsFunc()()

	repo.AssertExpectations(t)
}

func TestFinalizeReportsFunc_UnfinalizedQueryError(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() { container.Global = oldGlobal })

	repo := new(providertest.VisitDailyReportRepositoryMock)
	repo.On("UnfinalizedDaysBefore", mock.Anything).Return(nil, errors.New("boom"))

	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })
	container.MustSingleton(container.Global, func() provider.VisitDailyReportRepository { return repo })

	FinalizeReportsFunc()()

	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "FinalizeDay", mock.Anything)
}
