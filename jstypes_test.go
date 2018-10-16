package chik

import (
	"encoding/json"
	"testing"
	"time"
)

type Data struct {
	Time JSONTime
}

type testdata struct {
	arr   JSIntArr
	bytes string
}

var jsIntArrTestData = []testdata{testdata{[]CommandType{1}, "1"}, testdata{[]CommandType{1, 2}, "[1,2]"}, testdata{[]CommandType{}, "null"}}

func TestMarshalTime(t *testing.T) {
	expected := "\"10:02\""
	time := JSONTime{time.Date(2018, 01, 01, 10, 02, 00, 0, time.Local)}
	actual, err := json.Marshal(&time)
	if err != nil {
		t.Error(err)
	}
	if string(actual) != expected {
		t.Error("Time comparison failed: ", string(actual), expected)
	}
}

func TestMarshalIntArray(t *testing.T) {
	for _, d := range jsIntArrTestData {
		actual, err := json.Marshal(d.arr)
		if err != nil {
			t.Error(err)
		}
		if string(actual) != d.bytes {
			t.Error("Comparsion failed: ", string(actual), d.bytes)
		}
	}
}

func TestUnmarshalTime(t *testing.T) {
	actual := JSONTime{}
	err := json.Unmarshal([]byte("\"10:00\""), &actual)
	if err != nil {
		t.Error(err)
	}
	if actual.Hour() != 10 || actual.Minute() != 0 {
		t.Errorf("Error: expected 10:00 got %d:%d", actual.Hour(), actual.Minute())
	}
}

func TestWithStruct(t *testing.T) {
	actual := Data{}
	err := json.Unmarshal([]byte("{\"Time\": \"10:12\"}"), &actual)
	if err != nil {
		t.Error(err)
	}
	if actual.Time.Hour() != 10 || actual.Time.Minute() != 12 {
		t.Errorf("Error: expected 10:12 got %d:%d", actual.Time.Hour(), actual.Time.Minute())
	}
}

func TestUnmarshalJsIntArr(t *testing.T) {
	for _, d := range jsIntArrTestData {
		actual := JSIntArr{}
		err := json.Unmarshal([]byte(d.bytes), &actual)
		if err != nil {
			t.Error(err)
		}
		if len(d.arr) != len(actual) {
			t.Error("Expected: ", d.arr, " got: ", actual)
		}
		for i, v := range d.arr {
			if v != actual[i] {
				t.Error("Expected: ", d.arr, " got: ", actual)
			}
		}
	}
}
