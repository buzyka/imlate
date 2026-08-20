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

// newTestAggregator returns a started aggregator with a flush interval long
// enough that the ticker never fires during a test: every test drives the flush
// deterministically through Stop().
func newTestAggregator(repo *providertest.VisitDailyReportRepositoryMock) *Aggregator {
	agg := &Aggregator{Repo: repo, Logger: zap.NewNop().Sugar(), FlushInterval: time.Hour}
	agg.Start()
	return agg
}

func TestAggregator_ProcessesJob(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 30, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(nil)
	repo.On("RecalculateVisitorDay", int32(25), day).Return(nil)

	agg := newTestAggregator(repo)
	agg.Enqueue(25, day)
	agg.Stop() // flushes the outstanding set and waits

	repo.AssertExpectations(t)
}

func TestAggregator_EnsureDayRowsError_StillRecalculates(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 30, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(errors.New("boom"))
	// The visitor's own row is still upserted by the recalculation, so a failed
	// day initialization only costs the empty not_signed rows.
	repo.On("RecalculateVisitorDay", int32(25), day).Return(nil)

	agg := newTestAggregator(repo)
	agg.Enqueue(25, day)
	agg.Stop()

	repo.AssertExpectations(t)
}

// A failed EnsureDayRows must not be cached, so the next flush retries it.
func TestAggregator_EnsureDayRowsError_RetriedOnNextFlush(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 30, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(errors.New("boom")).Twice()
	repo.On("RecalculateVisitorDay", int32(25), day).Return(nil)

	agg := newTestAggregator(repo)
	agg.Enqueue(25, day)
	agg.flush()
	agg.Enqueue(25, day)
	agg.Stop()

	repo.AssertExpectations(t)
}

func TestAggregator_RecalculateError_Logged(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 30, 0, 0, time.UTC)
	repo.On("EnsureDayRows", day).Return(nil)
	repo.On("RecalculateVisitorDay", int32(25), day).Return(errors.New("boom"))

	agg := newTestAggregator(repo)
	agg.Enqueue(25, day)
	agg.Stop()

	repo.AssertExpectations(t)
}

func TestAggregator_EnqueueBeforeStart_NoOp(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	agg := &Aggregator{Repo: repo}
	// Not started: the dirty set is not accepting work, so this must not panic
	// or reach the repository.
	agg.Enqueue(1, time.Now())
	repo.AssertNotCalled(t, "EnsureDayRows", mock.Anything)
}

func TestAggregator_EnqueueOnNilReceiver_NoOp(t *testing.T) {
	var agg *Aggregator
	assert.NotPanics(t, func() { agg.Enqueue(1, time.Now()) })
}

func TestAggregator_StartAndStopAreIdempotent(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	agg := &Aggregator{Repo: repo, FlushInterval: time.Hour}
	agg.Start()
	agg.Start() // second call is a no-op
	agg.Stop()
	agg.Stop() // second call is a no-op
}

// Repeat events for the same visitor and day collapse into one recomputation —
// this is what keeps the morning sign-in burst from becoming a write storm.
func TestAggregator_CoalescesRepeatEventsForSameVisitorDay(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	repo.On("EnsureDayRows", mock.Anything).Return(nil).Once()
	repo.On("RecalculateVisitorDay", int32(7), day).Return(nil).Once()

	agg := newTestAggregator(repo)
	for i := 0; i < 10; i++ {
		agg.Enqueue(7, day)
	}
	agg.Stop()

	repo.AssertExpectations(t)
}

// Distinct visitors on the same day each get recomputed, but the day is
// initialized only once.
func TestAggregator_EnsureDayRowsOncePerDayAcrossVisitors(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	repo.On("EnsureDayRows", mock.Anything).Return(nil).Once()
	repo.On("RecalculateVisitorDay", int32(1), mock.Anything).Return(nil).Once()
	repo.On("RecalculateVisitorDay", int32(2), mock.Anything).Return(nil).Once()

	agg := newTestAggregator(repo)
	agg.Enqueue(1, day.Add(8*time.Hour))
	agg.Enqueue(2, day.Add(9*time.Hour))
	agg.Stop()

	repo.AssertExpectations(t)
}

// The ensured-days cache survives across flushes, so a day is initialized once
// no matter how many batches its events arrive in.
func TestAggregator_EnsureDayRowsCachedAcrossFlushes(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	repo.On("EnsureDayRows", mock.Anything).Return(nil).Once()
	repo.On("RecalculateVisitorDay", mock.Anything, mock.Anything).Return(nil).Times(3)

	agg := newTestAggregator(repo)
	agg.Enqueue(1, day.Add(8*time.Hour))
	agg.flush()
	agg.Enqueue(2, day.Add(9*time.Hour))
	agg.flush()
	agg.Enqueue(3, day.Add(10*time.Hour))
	agg.Stop()

	repo.AssertExpectations(t)
}

// Different days are initialized independently.
func TestAggregator_EnsureDayRowsPerDistinctDay(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	d1 := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	repo.On("EnsureDayRows", d1).Return(nil).Once()
	repo.On("EnsureDayRows", d2).Return(nil).Once()
	repo.On("RecalculateVisitorDay", int32(1), d1).Return(nil).Once()
	repo.On("RecalculateVisitorDay", int32(1), d2).Return(nil).Once()

	agg := newTestAggregator(repo)
	agg.Enqueue(1, d1)
	agg.Enqueue(1, d2)
	agg.Stop()

	repo.AssertExpectations(t)
}

func TestAggregator_FlushOnEmptySetIsNoOp(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	agg := newTestAggregator(repo)
	agg.flush()
	agg.Stop()

	repo.AssertNotCalled(t, "EnsureDayRows", mock.Anything)
	repo.AssertNotCalled(t, "RecalculateVisitorDay", mock.Anything, mock.Anything)
}

// The ticker path drives a flush without any Stop().
func TestAggregator_FlushesOnTicker(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	done := make(chan struct{})
	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("RecalculateVisitorDay", int32(4), day).
		Run(func(mock.Arguments) { close(done) }).
		Return(nil)

	agg := &Aggregator{Repo: repo, Logger: zap.NewNop().Sugar(), FlushInterval: 5 * time.Millisecond}
	agg.Start()
	t.Cleanup(agg.Stop)
	agg.Enqueue(4, day)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ticker did not trigger a flush")
	}
}

func TestAggregator_BacklogFull_DropsWithoutBlocking(t *testing.T) {
	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	agg := &Aggregator{Repo: repo, Logger: zap.NewNop().Sugar(), FlushInterval: time.Hour}
	agg.pending = make(map[job]struct{}, maxPendingEntries)
	agg.ensuredDays = make(map[string]time.Time)
	agg.started = true
	for i := 0; i < maxPendingEntries; i++ {
		agg.pending[job{visitorID: int32(i), day: day}] = struct{}{}
	}

	agg.Enqueue(999999, day) // must be dropped and must not block
	assert.Len(t, agg.pending, maxPendingEntries)
	assert.NotContains(t, agg.pending, job{visitorID: 999999, day: day})
}

func TestAggregator_MarkDayEnsured_InitialisesNilCache(t *testing.T) {
	agg := &Aggregator{}
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	agg.markDayEnsured(day)

	assert.True(t, agg.dayEnsured(day))
}

// The ensured-days cache must not grow without bound as days go by.
func TestAggregator_EnsuredDaysCacheEvictsOldEntries(t *testing.T) {
	agg := &Aggregator{ensuredDays: make(map[string]time.Time)}
	old := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	agg.markDayEnsured(old)
	assert.True(t, agg.dayEnsured(old))

	agg.markDayEnsured(recent)
	assert.False(t, agg.dayEnsured(old), "day beyond the retention window should be evicted")
	assert.True(t, agg.dayEnsured(recent))
}

// Jobs enqueued for the same calendar day but with different event timestamps
// must still initialize the day only once.
func TestDistinctDays_CollapsesByCalendarDay(t *testing.T) {
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	batch := map[job]struct{}{
		{visitorID: 1, day: day.Add(8 * time.Hour)}:  {},
		{visitorID: 2, day: day.Add(13 * time.Hour)}: {},
		{visitorID: 3, day: day.AddDate(0, 0, 1)}:    {},
	}

	days := distinctDays(batch)
	assert.Len(t, days, 2)
}
