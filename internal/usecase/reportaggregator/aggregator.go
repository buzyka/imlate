// Package reportaggregator provides an in-process background worker that keeps
// the aggregated visit_daily_report table up to date. Tracking handlers mark a
// (visitor, day) pair dirty after storing an event; the worker flushes the dirty
// set on a short interval, ensuring the day's rows exist and recomputing each
// dirty visitor's row. Updates are best-effort: the daily finalization cron
// reconciles anything missed.
//
// Work is coalesced rather than queued one-job-per-event. Recalculation reads
// the whole day for a visitor and rewrites the row, so N events for the same
// visitor on the same day produce an identical result — collapsing them into a
// single recomputation turns the morning sign-in burst into a steady trickle of
// database writes instead of one write storm per event.
package reportaggregator

import (
	"sync"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
	"go.uber.org/zap"
)

const (
	// defaultFlushInterval is how often the dirty set is drained. Short enough
	// that reports for the current day stay effectively live, long enough to
	// coalesce the events of a sign-in rush.
	defaultFlushInterval = 2 * time.Second
	// maxPendingEntries bounds the dirty set. It cannot normally be reached
	// (the set holds at most one entry per visitor per day), so hitting it means
	// the database is unreachable and flushes are failing.
	maxPendingEntries = 10000
	// ensuredDayRetention is how long a day stays in the ensured-days cache.
	// Two days covers "yesterday" for late events without growing unbounded.
	ensuredDayRetention = 48 * time.Hour

	dayLayout = "2006-01-02"
)

type job struct {
	visitorID int32
	day       time.Time
}

// Aggregator consumes visit-report update jobs on a background goroutine.
type Aggregator struct {
	Repo   provider.VisitDailyReportRepository `container:"type"`
	Logger *zap.SugaredLogger                  `container:"type"`

	// FlushInterval overrides how often the dirty set is drained. Zero means
	// defaultFlushInterval. Must be set before Start.
	FlushInterval time.Duration

	mu      sync.Mutex
	pending map[job]struct{}
	started bool
	// ensuredDays remembers which days EnsureDayRows already handled, so the
	// day-initialization query runs once per day instead of once per event.
	ensuredDays map[string]time.Time

	done      chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once
}

// Start launches the background worker. It is safe to call multiple times; only
// the first call has an effect.
func (a *Aggregator) Start() {
	a.startOnce.Do(func() {
		a.mu.Lock()
		if a.pending == nil {
			a.pending = make(map[job]struct{})
		}
		if a.ensuredDays == nil {
			a.ensuredDays = make(map[string]time.Time)
		}
		a.started = true
		a.mu.Unlock()

		a.done = make(chan struct{})
		a.wg.Add(1)
		go a.run()
	})
}

func (a *Aggregator) run() {
	defer a.wg.Done()

	interval := a.FlushInterval
	if interval <= 0 {
		interval = defaultFlushInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.flush()
		case <-a.done:
			a.flush()
			return
		}
	}
}

// Enqueue marks the given visitor and day for background recalculation. It never
// blocks the caller and collapses repeat calls for the same (visitor, day) into
// a single recomputation.
func (a *Aggregator) Enqueue(visitorID int32, day time.Time) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return
	}
	if len(a.pending) >= maxPendingEntries {
		if a.Logger != nil {
			a.Logger.Warnw("visit report aggregation backlog full, dropping job",
				"visitor_id", visitorID, "day", day.Format(dayLayout), "pending", len(a.pending))
		}
		return
	}
	a.pending[job{visitorID: visitorID, day: day}] = struct{}{}
}

// flush takes the current dirty set and writes it out. Day initialization runs
// at most once per distinct day; each dirty (visitor, day) pair is recomputed
// once regardless of how many events produced it.
func (a *Aggregator) flush() {
	a.mu.Lock()
	batch := a.pending
	if len(batch) == 0 {
		a.mu.Unlock()
		return
	}
	a.pending = make(map[job]struct{})
	a.mu.Unlock()

	for _, day := range distinctDays(batch) {
		if a.dayEnsured(day) {
			continue
		}
		if err := a.Repo.EnsureDayRows(day); err != nil {
			if a.Logger != nil {
				a.Logger.Errorw("visit report aggregation failed",
					"op", "ensure day rows", "day", day.Format(dayLayout), "error", err)
			}
			// Recalculation still upserts the acting visitor's own row, so the
			// day is only missing its empty not_signed rows until the next
			// attempt or the nightly finalization.
			continue
		}
		a.markDayEnsured(day)
	}

	for j := range batch {
		if err := a.Repo.RecalculateVisitorDay(j.visitorID, j.day); err != nil {
			a.logError("recalculate visitor day", j, err)
		}
	}
}

// distinctDays returns the unique days present in a batch. Jobs carry the full
// event timestamp, so several jobs on the same calendar day can hold different
// time.Time values; they are keyed by formatted date to collapse correctly.
func distinctDays(batch map[job]struct{}) []time.Time {
	seen := make(map[string]struct{}, len(batch))
	days := make([]time.Time, 0, len(batch))
	for j := range batch {
		key := j.day.Format(dayLayout)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		days = append(days, j.day)
	}
	return days
}

func (a *Aggregator) dayEnsured(day time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.ensuredDays[day.Format(dayLayout)]
	return ok
}

func (a *Aggregator) markDayEnsured(day time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ensuredDays == nil {
		a.ensuredDays = make(map[string]time.Time)
	}
	a.ensuredDays[day.Format(dayLayout)] = day
	for key, d := range a.ensuredDays {
		if day.Sub(d) > ensuredDayRetention {
			delete(a.ensuredDays, key)
		}
	}
}

// Stop signals the worker to flush the outstanding dirty set and waits for it to
// exit. It is safe to call multiple times.
func (a *Aggregator) Stop() {
	a.stopOnce.Do(func() {
		if a.done != nil {
			close(a.done)
			a.wg.Wait()
		}
		a.mu.Lock()
		a.started = false
		a.mu.Unlock()
	})
}

func (a *Aggregator) logError(op string, j job, err error) {
	if a.Logger != nil {
		a.Logger.Errorw("visit report aggregation failed",
			"op", op, "visitor_id", j.visitorID, "day", j.day.Format(dayLayout), "error", err)
	}
}
