package entity

import (
	"errors"
	"sync"
	"time"
)

// MaxFireAlarmDuration caps how long one alarm may keep the public evacuation
// roster readable. The roster names every child in a class and says who is in
// the building, so the window has to be short by construction: a typo in the
// duration, or an admin who forgets to switch the alarm off after a drill,
// must not leave that data exposed indefinitely.
const MaxFireAlarmDuration = 24 * time.Hour

var (
	// ErrDurationRequired is returned when an alarm is enabled without a
	// positive duration.
	ErrDurationRequired = errors.New("fire alarm duration is required and must be positive")
	// ErrDurationTooLong is returned when the requested duration exceeds
	// MaxFireAlarmDuration.
	ErrDurationTooLong = errors.New("fire alarm duration exceeds the maximum allowed")
)

// FireAlarmState is the process-wide switch that decides whether the public
// evacuation roster returns data. An admin turns it on for a bounded window;
// once that window elapses the state lapses on its own, so nobody has to
// remember to turn it off.
//
// The state is intentionally process-local and not persisted: a restart leaves
// the alarm off, which fails in the safe direction — the roster goes dark
// rather than staying open.
//
// The zero value is a usable, switched-off alarm, so the DI container can
// create it as &FireAlarmState{} with no constructor.
type FireAlarmState struct {
	mu        sync.Mutex
	enabled   bool
	startedAt time.Time
	duration  time.Duration

	// now is the clock used for expiry. nil means time.Now. Tests inject a
	// fake, following the pattern of theme.Service.clock().
	now func() time.Time
}

// FireAlarmSnapshot is a consistent read of the alarm, taken under the lock.
type FireAlarmSnapshot struct {
	Enabled   bool
	StartedAt time.Time
	Duration  time.Duration
	Remaining time.Duration
}

// Enable switches the alarm on for the given duration, starting now.
func (s *FireAlarmState) Enable(duration time.Duration) error {
	if duration <= 0 {
		return ErrDurationRequired
	}
	if duration > MaxFireAlarmDuration {
		return ErrDurationTooLong
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.enabled = true
	s.startedAt = s.clock()
	s.duration = duration
	return nil
}

// Disable switches the alarm off immediately and clears the timing metadata.
func (s *FireAlarmState) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.enabled = false
	s.startedAt = time.Time{}
	s.duration = 0
}

// IsActive reports whether the alarm is on and still within its window,
// expiring it in place if the window has elapsed.
//
// A nil receiver reports false. This guards the one direction that matters: a
// miswired container must leave the public roster closed, never open it.
func (s *FireAlarmState) IsActive() bool {
	if s == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.evaluate(s.clock())
}

// Snapshot returns the current state, expiring the alarm first so a caller can
// never be told an elapsed alarm is still running.
func (s *FireAlarmState) Snapshot() FireAlarmSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	// One clock read shared with the expiry check. Reading it twice would let
	// the alarm lapse between the two reads and yield a negative remaining.
	now := s.clock()
	if !s.evaluate(now) {
		return FireAlarmSnapshot{}
	}

	return FireAlarmSnapshot{
		Enabled:   true,
		StartedAt: s.startedAt,
		Duration:  s.duration,
		Remaining: s.duration - now.Sub(s.startedAt),
	}
}

// evaluate is the lazy expiry check, against the caller's clock reading. It
// must be called with s.mu held, and it writes (clearing enabled), which is why
// this type uses a plain Mutex rather than an RWMutex — Go cannot upgrade a
// read lock to a write lock.
func (s *FireAlarmState) evaluate(now time.Time) bool {
	if !s.enabled {
		return false
	}
	if now.Sub(s.startedAt) >= s.duration {
		s.enabled = false
		s.startedAt = time.Time{}
		s.duration = 0
		return false
	}
	return true
}

// clock returns the time source for expiry. It must be called with s.mu held.
//
// This deliberately does NOT use util.Now(), the app clock used elsewhere.
// util.Now() freezes to a fixed hour and minute once the public POST
// /change-time endpoint has been called, so two successive calls return the
// identical instant and now.Sub(startedAt) is always zero — an alarm built on
// it would never expire, and the rosters it guards would stay public forever.
// Expiry here is a safety control, so it runs on the real clock.
func (s *FireAlarmState) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}
