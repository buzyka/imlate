package util

import "time"

type timeStruct struct {
	hour int
	minute int
}

var currentTimeStruct *timeStruct

func SetCurrentTime(hour, minute int) {
	currentTimeStruct = &timeStruct{
		hour: hour,
		minute: minute,
	}
}

func Now() time.Time {
	if currentTimeStruct != nil {
		now := time.Now()
		return time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			currentTimeStruct.hour,
			currentTimeStruct.minute,
			0, 0,
			now.Location(),
		)
	}
	return time.Now()
}

func ParseTimeStrToLocal(timeStr string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(time.Local), nil
}

func FromLocalTimeToTimeStr(t time.Time, loc *time.Location) string {
	return t.In(loc).Format(time.RFC3339)
}
