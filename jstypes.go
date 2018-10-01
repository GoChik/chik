package chik

import (
	"encoding/json"
	"fmt"
	"time"
)

// JSIntArr an int array that can be rapresented in javascript like an integer or like an array of integers
type JSIntArr []CommandType

func (t JSIntArr) MarshalJSON() ([]byte, error) {
	switch len(t) {
	case 0:
		return []byte("null"), nil
	case 1:
		return json.Marshal(t[0])
	}
	return json.Marshal([]CommandType(t))
}

func (t *JSIntArr) UnmarshalJSON(data []byte) error {
	var arrResult JSIntArr
	err := json.Unmarshal(data, (*[]CommandType)(&arrResult))
	if err == nil {
		*t = arrResult
		return nil
	}

	var intResult CommandType
	err = json.Unmarshal(data, &intResult)
	if err == nil {
		*t = JSIntArr{intResult}
	}
	return err
}

// JSONTime a time specialization that allows a compact string rapresentation: hh:mm
type JSONTime struct {
	time.Time
}

// MarshalJSON returns the js string that represent the current hours and minutes
func (j *JSONTime) MarshalJSON() ([]byte, error) {
	if j.IsZero() {
		return []byte(""), nil
	}

	timeString := fmt.Sprintf("\"%s\"", j.Format("15:04"))
	return []byte(timeString), nil
}

// UnmarshalJSON returns the time corresponding to the string "15:04"
func (j *JSONTime) UnmarshalJSON(data []byte) error {
	parsedTime, err := time.Parse("\"15:04\"", string(data))
	if err == nil {
		parsedTime = parsedTime.Add(24 * time.Hour)
		*j = JSONTime{parsedTime}
	}
	return err
}
