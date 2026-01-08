package entity

import "time"

type Schedule struct {
	Periods map[int32]*RegistrationPeriod
}

type RegistrationPeriod struct {
	ID     int32
	Name   string
	Finish time.Time
	Start  time.Time
	Time   time.Time
}

func (s *Schedule) AddPeriod(period *RegistrationPeriod) {
	if s.Periods == nil {
		s.Periods = make(map[int32]*RegistrationPeriod)
	}
	s.Periods[period.ID] = period
}

func (s *Schedule) GetPeriodByName(name string) (*RegistrationPeriod, bool) {
	for _, period := range s.Periods {
		if period.Name == name {
			return period, true
		}
	}
	return nil, false
}

func (s *Schedule) GetPeriodByTime(currentTime time.Time) (*RegistrationPeriod, bool) {
	if firstPeriod, ok := s.FirstPeriod(); ok {
		if currentTime.Before(firstPeriod.Start) {
			return firstPeriod, true
		}
	}

	for _, period := range s.Periods {
		if currentTime.After(period.Start) && currentTime.Before(period.Finish) {
			return period, true
		}
		// Include boundary conditions
		if !currentTime.Before(period.Start) && !currentTime.After(period.Finish) {
			return period, true
		}
	}
	return nil, false
}

func (s *Schedule) FirstPeriod() (*RegistrationPeriod, bool) {
	var first *RegistrationPeriod
	for _, period := range s.Periods {
		if first == nil || period.Start.Before(first.Start) {
			first = period
		}
	}
	if first != nil {
		return first, true
	}
	return nil, false
}

func (s *Schedule) LastPeriod() (*RegistrationPeriod, bool) {
	var last *RegistrationPeriod
	for _, period := range s.Periods {
		if last == nil || period.Finish.After(last.Finish) {
			last = period
		}
	}
	if last != nil {
		return last, true
	}
	return nil, false
}

func (s *Schedule) IsBeforeFirstPeriod(currentTime time.Time) bool {
	if firstPeriod, ok := s.FirstPeriod(); ok {
		return currentTime.Before(firstPeriod.Start)
	}
	return false
}

func (s *Schedule) IsAfterLastPeriod(currentTime time.Time) bool {
	if lastPeriod, ok := s.LastPeriod(); ok {
		return currentTime.After(lastPeriod.Finish)
	}
	return false
}
