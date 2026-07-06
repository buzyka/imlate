package reportaggregator

import (
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestAggregator_ProcessesJob(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 30, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(nil)
	repo.On("RecalculateVisitorDay", int32(25), day).Return(nil)

	agg := &Aggregator{Repo: repo}
	agg.Start()
	agg.Enqueue(25, day)
	agg.Stop() // drains and waits

	repo.AssertExpectations(t)
}

func TestAggregator_EnsureDayRowsError_SkipsRecalc(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 30, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(errors.New("boom"))

	agg := &Aggregator{Repo: repo}
	agg.Start()
	agg.Enqueue(25, day)
	agg.Stop()

	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "RecalculateVisitorDay", mock.Anything, mock.Anything)
}

func TestAggregator_RecalculateError_Logged(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 30, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(nil)
	repo.On("RecalculateVisitorDay", int32(25), day).Return(errors.New("boom"))

	agg := &Aggregator{Repo: repo, Logger: zap.NewNop().Sugar()}
	agg.Start()
	agg.Enqueue(25, day)
	agg.Stop()

	repo.AssertExpectations(t)
}

func TestAggregator_EnqueueBeforeStart_NoOp(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	agg := &Aggregator{Repo: repo}
	// Not started: jobs channel is nil, so this must not panic or call the repo.
	agg.Enqueue(1, time.Now())
	repo.AssertNotCalled(t, "EnsureDayRows", mock.Anything)
}

func TestAggregator_StartAndStopAreIdempotent(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	agg := &Aggregator{Repo: repo}
	agg.Start()
	agg.Start() // second call is a no-op
	agg.Stop()
	agg.Stop() // second call is a no-op
}

func TestAggregator_Drain_ProcessesQueuedJobs(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(nil)
	repo.On("RecalculateVisitorDay", int32(9), day).Return(nil)

	agg := &Aggregator{Repo: repo}
	agg.jobs = make(chan job, 4)
	agg.jobs <- job{visitorID: 9, day: day}

	agg.drain() // processes the queued job, then returns on empty

	repo.AssertExpectations(t)
}

func TestAggregator_QueueFull_DropsWithoutBlocking(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	repo.On("EnsureDayRows", mock.Anything).Return(nil).Maybe()
	repo.On("RecalculateVisitorDay", mock.Anything, mock.Anything).Return(nil).Maybe()

	agg := &Aggregator{Repo: repo, Logger: zap.NewNop().Sugar()}
	// Pre-fill the queue without a running worker so Enqueue hits the default branch.
	agg.jobs = make(chan job, 1)
	agg.jobs <- job{visitorID: 1, day: day}
	// This one must be dropped (queue full) and must not block.
	agg.Enqueue(2, day)
	assert.Len(t, agg.jobs, 1)
}
