package util

import "time"

func ParseTimeStrToLocal(timeStr string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(time.Local), nil
}
