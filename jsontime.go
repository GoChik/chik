package iosomething

import (
	"fmt"
	"time"
)

// JSONTime a time specialization that allows a compact string rapresentation: hh:mm
type JSONTime time.Time

// MarshalJSON returns the js string that represent the current hours and minutes
func (j *JSONTime) MarshalJSON() ([]byte, error) {
	if time.Time(*j).IsZero() {
		return []byte(""), nil
	}

	timeString := fmt.Sprintf("\"%s\"", time.Time(*j).Format("15:04"))
	return []byte(timeString), nil
}

// UnmarshalJSON returns the time corresponding to the string "15:04"
func (j *JSONTime) UnmarshalJSON(data []byte) error {
	parsedTime, err := time.Parse("\"15:04\"", string(data))
	if err == nil {
		parsedTime = parsedTime.Add(24 * time.Hour)
		*j = JSONTime(parsedTime)
	}
	return err
}
