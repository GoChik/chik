package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/mitchellh/mapstructure"
)

// JSIntArr an int array that can be rapresented in javascript like an integer or like an array of integers
type JSIntArr []Action

func (t JSIntArr) MarshalJSON() ([]byte, error) {
	switch len(t) {
	case 0:
		return []byte("null"), nil
	case 1:
		return json.Marshal(t[0])
	}
	return json.Marshal([]Action(t))
}

func (t *JSIntArr) UnmarshalJSON(data []byte) error {
	var arrResult JSIntArr
	err := json.Unmarshal(data, (*[]Action)(&arrResult))
	if err == nil {
		*t = arrResult
		return nil
	}

	var intResult Action
	err = json.Unmarshal(data, &intResult)
	if err == nil {
		*t = JSIntArr{intResult}
	}
	return err
}

func IntToJsIntArr(sourceType, targetType reflect.Type, sourceData interface{}) (interface{}, error) {
	if sourceType.Kind() != reflect.Float64 {
		return sourceData, nil
	}

	if targetType != reflect.TypeOf(JSIntArr{}) {
		return sourceData, nil
	}

	return JSIntArr{Action(sourceData.(float64))}, nil
}

// JSONTime a time specialization that allows a compact string rapresentation: hh:mm
type JSONTime struct {
	time.Time
}

// MarshalJSON returns the js string that represent the current hours and minutes
func (j JSONTime) MarshalJSON() ([]byte, error) {
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

func StringToJsonTime(sourceType, targetType reflect.Type, sourceData interface{}) (interface{}, error) {
	if sourceType.Kind() != reflect.String {
		return sourceData, nil
	}

	if targetType != reflect.TypeOf(JSONTime{}) {
		return sourceData, nil
	}

	parsedTime, err := time.Parse("15:04", sourceData.(string))
	if err == nil {
		parsedTime = parsedTime.Add(24 * time.Hour)
	}
	return JSONTime{parsedTime}, err
}

func StringInterfaceToJsonRawMessage(sourceType, targetType reflect.Type, sourceData interface{}) (interface{}, error) {
	if sourceType.Kind() != reflect.Map {
		return sourceData, nil
	}

	if targetType != reflect.TypeOf(json.RawMessage{}) {
		return sourceData, nil
	}

	data, err := json.Marshal(sourceData)
	return json.RawMessage(data), err
}

func Decode(input, output interface{}, hooks ...mapstructure.DecodeHookFunc) error {
	hooks = append(hooks, IntToJsIntArr, StringToJsonTime, StringInterfaceToJsonRawMessage)
	log.Debug("HOOKS", len(hooks))
	config := mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(hooks...),
		Result:           output,
	}
	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}
