// Package reportaggregator provides an in-process background worker that keeps
// the aggregated visit_daily_report table up to date. Tracking handlers enqueue
// a (visitor, day) job after storing an event; the worker ensures the day's
// rows exist and recomputes the acting visitor's row. Updates are best-effort:
// the daily finalization cron reconciles anything missed.
package reportaggregator

import (
	"sync"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
	"go.uber.org/zap"
)

const (
	defaultQueueSize = 256
	dayLayout        = "2006-01-02"
)

type job struct {
	visitorID int32
	day       time.Time
}

// Aggregator consumes visit-report update jobs on a background goroutine.
type Aggregator struct {
	Repo   provider.VisitDailyReportRepository `container:"type"`
	Logger *zap.SugaredLogger                  `container:"type"`

	jobs      chan job
	done      chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once
}

// Start launches the background worker. It is safe to call multiple times; only
// the first call has an effect.
func (a *Aggregator) Start() {
	a.startOnce.Do(func() {
		if a.jobs == nil {
			a.jobs = make(chan job, defaultQueueSize)
		}
		a.done = make(chan struct{})
		a.wg.Add(1)
		go a.run()
	})
}

func (a *Aggregator) run() {
	defer a.wg.Done()
	for {
		select {
		case j := <-a.jobs:
			a.process(j)
		case <-a.done:
			a.drain()
			return
		}
	}
}

// drain processes any jobs already queued, then returns.
func (a *Aggregator) drain() {
	for {
		select {
		case j := <-a.jobs:
			a.process(j)
		default:
			return
		}
	}
}

func (a *Aggregator) process(j job) {
	if err := a.Repo.EnsureDayRows(j.day); err != nil {
		a.logError("ensure day rows", j, err)
		return
	}
	if err := a.Repo.RecalculateVisitorDay(j.visitorID, j.day); err != nil {
		a.logError("recalculate visitor day", j, err)
	}
}

// Enqueue schedules a background recalculation for the given visitor and day.
// It never blocks the caller: if the queue is full the job is dropped (the
// daily finalization cron will reconcile it).
func (a *Aggregator) Enqueue(visitorID int32, day time.Time) {
	if a.jobs == nil {
		return
	}
	select {
	case a.jobs <- job{visitorID: visitorID, day: day}:
	default:
		if a.Logger != nil {
			a.Logger.Warnw("visit report aggregation queue full, dropping job",
				"visitor_id", visitorID, "day", day.Format(dayLayout))
		}
	}
}

// Stop signals the worker to finish draining queued jobs and waits for it to
// exit. It is safe to call multiple times.
func (a *Aggregator) Stop() {
	a.stopOnce.Do(func() {
		if a.done != nil {
			close(a.done)
			a.wg.Wait()
		}
	})
}

func (a *Aggregator) logError(op string, j job, err error) {
	if a.Logger != nil {
		a.Logger.Errorw("visit report aggregation failed",
			"op", op, "visitor_id", j.visitorID, "day", j.day.Format(dayLayout), "error", err)
	}
}
