package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSchedule_AddPeriod(t *testing.T) {
	s := &Schedule{}

	p1 := &RegistrationPeriod{ID: 1, Name: "P1"}
	s.AddPeriod(p1)

	assert.NotNil(t, s.Periods)
	assert.Len(t, s.Periods, 1)
	assert.Equal(t, p1, s.Periods[1])

	p2 := &RegistrationPeriod{ID: 2, Name: "P2"}
	s.AddPeriod(p2)

	assert.Len(t, s.Periods, 2)
	assert.Equal(t, p2, s.Periods[2])
}

func TestSchedule_GetPeriodByName(t *testing.T) {
	s := &Schedule{}
	p1 := &RegistrationPeriod{ID: 1, Name: "Period 1"}
	p2 := &RegistrationPeriod{ID: 2, Name: "Period 2"}

	s.AddPeriod(p1)
	s.AddPeriod(p2)

	t.Run("Found", func(t *testing.T) {
		p, ok := s.GetPeriodByName("Period 1")
		assert.True(t, ok)
		assert.Equal(t, p1, p)
	})

	t.Run("NotFound", func(t *testing.T) {
		p, ok := s.GetPeriodByName("Period 3")
		assert.False(t, ok)
		assert.Nil(t, p)
	})

	t.Run("EmptySchedule", func(t *testing.T) {
		empty := &Schedule{}
		p, ok := empty.GetPeriodByName("Period 1")
		assert.False(t, ok)
		assert.Nil(t, p)
	})
}

func TestSchedule_FirstPeriod(t *testing.T) {
	now := time.Now()
	p1 := &RegistrationPeriod{ID: 1, Start: now.Add(1 * time.Hour)}
	p2 := &RegistrationPeriod{ID: 2, Start: now.Add(2 * time.Hour)}
	p3 := &RegistrationPeriod{ID: 3, Start: now}

	s := &Schedule{}
	s.AddPeriod(p1)
	s.AddPeriod(p2)
	s.AddPeriod(p3)

	first, ok := s.FirstPeriod()
	assert.True(t, ok)
	assert.Equal(t, p3, first)

	empty := &Schedule{}
	first, ok = empty.FirstPeriod()
	assert.False(t, ok)
	assert.Nil(t, first)
}

func TestSchedule_LastPeriod(t *testing.T) {
	now := time.Now()
	p1 := &RegistrationPeriod{ID: 1, Finish: now.Add(1 * time.Hour)}
	p2 := &RegistrationPeriod{ID: 2, Finish: now.Add(2 * time.Hour)}
	p3 := &RegistrationPeriod{ID: 3, Finish: now}

	s := &Schedule{}
	s.AddPeriod(p1)
	s.AddPeriod(p2)
	s.AddPeriod(p3)

	last, ok := s.LastPeriod()
	assert.True(t, ok)
	assert.Equal(t, p2, last)

	empty := &Schedule{}
	last, ok = empty.LastPeriod()
	assert.False(t, ok)
	assert.Nil(t, last)
}

func TestSchedule_GetPeriodByTime(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

	p1 := &RegistrationPeriod{
		ID:     1,
		Name:   "P1",
		Start:  baseTime,
		Finish: baseTime.Add(1 * time.Hour), // 10:00 - 11:00
	}
	p2 := &RegistrationPeriod{
		ID:     2,
		Name:   "P2",
		Start:  baseTime.Add(2 * time.Hour), // 12:00 - 13:00
		Finish: baseTime.Add(3 * time.Hour),
	}

	s := &Schedule{}
	s.AddPeriod(p1)
	s.AddPeriod(p2)

	t.Run("BeforeFirstPeriod", func(t *testing.T) {
		// Should return first period
		checkTime := baseTime.Add(-1 * time.Hour) // 09:00
		p, ok := s.GetPeriodByTime(checkTime)
		assert.True(t, ok)
		assert.Equal(t, p1, p)
	})

	t.Run("InsidePeriod1", func(t *testing.T) {
		checkTime := baseTime.Add(30 * time.Minute) // 10:30
		p, ok := s.GetPeriodByTime(checkTime)
		assert.True(t, ok)
		assert.Equal(t, p1, p)
	})

	t.Run("AtStartOfPeriod1", func(t *testing.T) {
		checkTime := baseTime // 10:00
		p, ok := s.GetPeriodByTime(checkTime)
		assert.True(t, ok)
		assert.Equal(t, p1, p)
	})

	t.Run("AtFinishOfPeriod1", func(t *testing.T) {
		checkTime := baseTime.Add(1 * time.Hour) // 11:00
		p, ok := s.GetPeriodByTime(checkTime)
		assert.True(t, ok)
		assert.Equal(t, p1, p)
	})

	t.Run("BetweenPeriods", func(t *testing.T) {
		checkTime := baseTime.Add(90 * time.Minute) // 11:30
		p, ok := s.GetPeriodByTime(checkTime)
		assert.False(t, ok)
		assert.Nil(t, p)
	})

	t.Run("InsidePeriod2", func(t *testing.T) {
		checkTime := baseTime.Add(150 * time.Minute) // 12:30
		p, ok := s.GetPeriodByTime(checkTime)
		assert.True(t, ok)
		assert.Equal(t, p2, p)
	})

	t.Run("AfterLastPeriod", func(t *testing.T) {
		checkTime := baseTime.Add(4 * time.Hour) // 14:00
		p, ok := s.GetPeriodByTime(checkTime)
		assert.False(t, ok)
		assert.Nil(t, p)
	})

	t.Run("EmptySchedule", func(t *testing.T) {
		empty := &Schedule{}
		p, ok := empty.GetPeriodByTime(baseTime)
		assert.False(t, ok)
		assert.Nil(t, p)
	})
}
