package cron

import (
	"context"
	"testing"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type stubERPFactory struct{}

func (s *stubERPFactory) NewClient(_ context.Context) (erp.Client, error) {
	return nil, nil
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
		ISAMSBaseURL:         "",
		ISAMSAPIClientID:     "",
		ISAMSAPIClientSecret: "",
	}

	stopFunc, err := RunCron(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, stopFunc)
	// Calling stop should not panic
	stopFunc()
}

func TestRunCron_WhenERPIntegrated_ReturnsStopFuncAndNoError(t *testing.T) {
	cfg := &config.Config{
		ISAMSBaseURL:         "https://example.com",
		ISAMSAPIClientID:     "client-id",
		ISAMSAPIClientSecret: "client-secret",
		ForceERPSyncOnStart:     false,
	}

	stopFunc, err := RunCron(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, stopFunc)
	stopFunc()
}

func TestRunCron_WhenERPIntegratedAndForceERPSyncOnStart_StartsWithoutError(t *testing.T) {
	testContainer := container.New()
	oldGlobal := container.Global
	container.Global = testContainer
	t.Cleanup(func() {
		container.Global = oldGlobal
	})

	syncCalled := false
	photosCalled := false

	// Setup minimal container dependencies
	container.MustSingleton(container.Global, func() *zap.SugaredLogger { return zap.NewNop().Sugar() })

	cfg := &config.Config{
		ISAMSBaseURL:         "https://example.com",
		ISAMSAPIClientID:     "client-id",
		ISAMSAPIClientSecret: "client-secret",
		ForceERPSyncOnStart:     true,
	}

	// We can't easily test that goroutine runs without more complex setup
	// but we can at least verify the function doesn't panic
	stopFunc, err := RunCron(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, stopFunc)
	stopFunc()

	// Verify the variables are unused (just to avoid linter warnings about unused variables)
	_ = syncCalled
	_ = photosCalled
}
