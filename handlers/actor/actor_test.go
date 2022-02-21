package actor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gochik/chik/handlers/io"
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/types"
)

func getTestState(date time.Time) *State {
	state := make(map[string]interface{})
	for k, v := range map[string]interface{}{
		"io": io.Status{
			"test": io.CurrentStatus{
				DeviceDescription: bus.DeviceDescription{
					ID:    "test",
					Kind:  bus.DigitalOutputDevice,
					State: true,
				},
				LastStateChange: types.TimeIndication(date.Unix()),
			},
		},
		"datetime": types.TimeIndication(date.Unix()),
	} {
		js, _ := json.Marshal(v)
		var data interface{}
		json.Unmarshal(js, &data)
		state[k] = data
	}
	return CreateState(state, state)
}

func TestGetFieldDescriptor(t *testing.T) {
	date := time.Now()
	s := getTestState(date)

	f, err := s.GetFieldDescriptor("time(datetime)")
	if err != nil {
		t.Error(err)
	}
	receivedDate := time.Unix(int64(f.value.(types.TimeIndication)), 0)
	if receivedDate.Hour() != date.Hour() || receivedDate.Minute() != date.Minute() {
		t.Errorf("Failed comparing times: %v of type %t and %v of type %T",
			f.value, f.value,
			date.Unix(), date.Unix())
	}
}

func TestGetFieldDescriptorDuration(t *testing.T) {
	s := getTestState(time.Now())

	f, err := s.GetFieldDescriptor("duration(`1h`)")
	if err != nil {
		t.Error(err)
	}
	receivedDuration, ok := f.value.(time.Duration)
	if !ok {
		t.Errorf("Failed decoding duration: got %v %T", f.value, f.value)
	}
	if receivedDuration != time.Duration(1*time.Hour) {
		t.Errorf("Wrong duration: got %v ", receivedDuration)
	}
}

func TestGetFieldOperation(t *testing.T) {
	date := time.Now()
	s := getTestState(date)

	f, err := s.GetFieldDescriptor("time(datetime) after duration(`1h`)")
	if err != nil {
		t.Error(err)
	}
	receivedDate := time.Unix(int64(f.value.(types.TimeIndication)), 0)
	if receivedDate.Hour() != date.Hour()+1 || receivedDate.Minute() != date.Minute() {
		t.Errorf("Failed comparing times: %v of type %t and %v of type %T",
			f.value, f.value,
			date.Unix(), date.Unix())
	}
}

type comparedata struct {
	first    string
	second   string
	op       string
	expected bool
}

func TestCompareTimes(t *testing.T) {
	date := time.Now()
	s := getTestState(date)

	for _, d := range []comparedata{
		{
			first:    "time(datetime) after duration(`1h`)",
			second:   "time(datetime)",
			op:       ">",
			expected: true,
		},
		{
			first:    "time(datetime) after duration(`1m`)",
			second:   "time(datetime)",
			op:       "==",
			expected: false,
		},
		{
			first:    "time(datetime)",
			second:   "time(io.test.last_state_change)",
			op:       "==",
			expected: true,
		},
		{
			first:    "time(datetime)",
			second:   "time(io.test.last_state_change) after duration(`1h10m`)",
			op:       "<",
			expected: true,
		},
		{
			first:    "time(datetime) after duration(`10m`)",
			second:   "time(io.test.last_state_change)",
			op:       "<",
			expected: false,
		},
	} {
		v1, err := s.GetFieldDescriptor(d.first)
		if err != nil {
			t.Error(err)
		}

		v2, err := s.GetFieldDescriptor(d.second)
		if err != nil {
			t.Error(err)
		}

		result, err := Compare(v1.value, v2.value, d.op)
		if err != nil {
			t.Error(err)
		}

		if result != d.expected {
			t.Errorf("Comapre %v %v %v failed: expecting %v, got %v", d.first, d.op, d.second, d.expected, result)
		}
	}

}
