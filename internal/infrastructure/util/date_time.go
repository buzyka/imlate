package util

import "time"

var currentTime time.Time

var loc *time.Location


func initLoc() {
	var err error
	loc, err = time.LoadLocation("Europe/Berlin")
	if err != nil {
		// fallback: UTC (but better fail fast in real app init)
		loc = time.UTC
	}
}

func GetLocation() *time.Location {
	if loc == nil {
		initLoc()
	}
	return loc
}


func SetCurrentTime(t time.Time) {
	currentTime = t
}

func GetCurrentTime() time.Time {
	if !currentTime.IsZero() {
		return currentTime
	}
	return time.Now().In(GetLocation())
}

