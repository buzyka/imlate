package entity

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestAlarm returns an alarm on a clock the test drives by hand, plus the
// knob to advance it.
func newTestAlarm() (*FireAlarmState, func(time.Duration)) {
	base := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	offset := time.Duration(0)
	alarm := &FireAlarmState{now: func() time.Time { return base.Add(offset) }}
	return alarm, func(d time.Duration) { offset += d }
}

// The DI container creates this with &FireAlarmState{} and no constructor, so
// the zero value has to be a working, switched-off alarm.
func TestFireAlarmState_ZeroValueIsInactive(t *testing.T) {
	var alarm FireAlarmState

	assert.False(t, alarm.IsActive())
	assert.Equal(t, FireAlarmSnapshot{}, alarm.Snapshot())
}

func TestFireAlarmState_EnableActivates(t *testing.T) {
	alarm, _ := newTestAlarm()

	require.NoError(t, alarm.Enable(time.Hour))

	assert.True(t, alarm.IsActive())
}

func TestFireAlarmState_StaysActiveInsideWindow(t *testing.T) {
	alarm, advance := newTestAlarm()
	require.NoError(t, alarm.Enable(time.Hour))

	advance(59 * time.Minute)

	assert.True(t, alarm.IsActive())
}

// The window is half-open: at exactly the duration the alarm is already over.
func TestFireAlarmState_ExpiresExactlyAtDuration(t *testing.T) {
	alarm, advance := newTestAlarm()
	require.NoError(t, alarm.Enable(time.Hour))

	advance(time.Hour)

	assert.False(t, alarm.IsActive())
}

// Expiry must be recorded, not recomputed: once lapsed, the alarm stays off
// even if the clock were to move backwards (NTP correction, DST, a test).
func TestFireAlarmState_ExpiryIsPersistedNotRecomputed(t *testing.T) {
	base := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	offset := time.Hour * 2
	alarm := &FireAlarmState{now: func() time.Time { return base.Add(offset) }}
	require.NoError(t, alarm.Enable(time.Hour))

	offset += time.Hour
	require.False(t, alarm.IsActive(), "precondition: the alarm has lapsed")

	offset -= 2 * time.Hour

	assert.False(t, alarm.IsActive(), "a rewound clock must not revive a lapsed alarm")
}

func TestFireAlarmState_DisableClearsTiming(t *testing.T) {
	alarm, _ := newTestAlarm()
	require.NoError(t, alarm.Enable(time.Hour))

	alarm.Disable()

	assert.False(t, alarm.IsActive())
	assert.Equal(t, FireAlarmSnapshot{}, alarm.Snapshot())
}

func TestFireAlarmState_EnableValidation(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantErr  error
	}{
		{name: "zero is rejected", duration: 0, wantErr: ErrDurationRequired},
		{name: "negative is rejected", duration: -time.Second, wantErr: ErrDurationRequired},
		{name: "one second is allowed", duration: time.Second},
		{name: "exactly the maximum is allowed", duration: MaxFireAlarmDuration},
		{
			name:     "one second over the maximum is rejected",
			duration: MaxFireAlarmDuration + time.Second,
			wantErr:  ErrDurationTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alarm, _ := newTestAlarm()

			err := alarm.Enable(tt.duration)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				// A rejected duration must leave the alarm off, not half-set.
				assert.False(t, alarm.IsActive())
				return
			}
			require.NoError(t, err)
			assert.True(t, alarm.IsActive())
		})
	}
}

func TestFireAlarmState_SnapshotReportsRemaining(t *testing.T) {
	alarm, advance := newTestAlarm()
	require.NoError(t, alarm.Enable(time.Hour))

	first := alarm.Snapshot()
	require.True(t, first.Enabled)
	assert.Equal(t, time.Hour, first.Duration)
	assert.Equal(t, time.Hour, first.Remaining)
	assert.False(t, first.StartedAt.IsZero())

	advance(20 * time.Minute)
	second := alarm.Snapshot()

	assert.Equal(t, 40*time.Minute, second.Remaining)
	assert.Equal(t, first.StartedAt, second.StartedAt, "the start must not drift")
}

// A lapsed alarm reports the zero snapshot rather than a negative remaining.
func TestFireAlarmState_SnapshotAfterExpiryIsZero(t *testing.T) {
	alarm, advance := newTestAlarm()
	require.NoError(t, alarm.Enable(time.Minute))

	advance(5 * time.Minute)

	assert.Equal(t, FireAlarmSnapshot{}, alarm.Snapshot())
}

// Enabling again while already running restarts the window rather than
// extending or ignoring it — an admin re-arming during a drill means "from now".
func TestFireAlarmState_EnableRestartsTheWindow(t *testing.T) {
	alarm, advance := newTestAlarm()
	require.NoError(t, alarm.Enable(time.Hour))

	advance(50 * time.Minute)
	require.NoError(t, alarm.Enable(time.Hour))

	assert.Equal(t, time.Hour, alarm.Snapshot().Remaining)
}

// CI runs the suite with -race; this is what makes that meaningful here. The
// alarm is read by every public page request and written by the admin API.
func TestFireAlarmState_ConcurrentAccess(t *testing.T) {
	var alarm FireAlarmState
	const goroutines = 8
	const iterations = 200

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				switch (n + j) % 4 {
				case 0:
					_ = alarm.Enable(time.Hour)
				case 1:
					alarm.Disable()
				case 2:
					_ = alarm.IsActive()
				default:
					_ = alarm.Snapshot()
				}
			}
		}(i)
	}
	wg.Wait()
}

// A container that failed to provide the alarm must read as closed, not panic
// and not — far worse — read as open.
func TestFireAlarmState_NilReceiverIsInactive(t *testing.T) {
	var alarm *FireAlarmState

	assert.NotPanics(t, func() {
		assert.False(t, alarm.IsActive())
	})
}
