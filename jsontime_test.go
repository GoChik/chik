package chik

import (
	"encoding/json"
	"testing"
	"time"
)

type Data struct {
	Time JSONTime
}

func TestMarshalTime(t *testing.T) {
	expected := "\"10:02\""
	time := JSONTime(time.Date(2018, 01, 01, 10, 02, 00, 0, time.UTC))
	actual, err := json.Marshal(&time)
	if err != nil {
		t.Error(err)
	}
	if string(actual) != expected {
		t.Error("Time comparison failed: ", string(actual), expected)
	}
}

func TestUnmarshalTime(t *testing.T) {
	actual := JSONTime{}
	err := json.Unmarshal([]byte("\"10:00\""), &actual)
	if err != nil {
		t.Error(err)
	}
	if time.Time(actual).Hour() != 10 || time.Time(actual).Minute() != 0 {
		t.Errorf("Error: expected 10:00 got %d:%d", time.Time(actual).Hour(), time.Time(actual).Minute())
	}
}

func TestWithStruct(t *testing.T) {
	actual := Data{}
	err := json.Unmarshal([]byte("{\"Time\": \"10:12\"}"), &actual)
	if err != nil {
		t.Error(err)
	}
	if time.Time(actual.Time).Hour() != 10 || time.Time(actual.Time).Minute() != 12 {
		t.Errorf("Error: expected 10:12 got %d:%d", time.Time(actual.Time).Hour(), time.Time(actual.Time).Minute())
	}
}
