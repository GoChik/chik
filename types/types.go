package types

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

type Comparable interface {
	Compare(other Comparable) (int8, error)
}

// TimeIndication defines a basic struct for time with resolution of 1 sec
type TimeIndication int64

func (t TimeIndication) Compare(other Comparable) (int8, error) {
	otherc, ok := other.(TimeIndication)
	if !ok {
		return 0, errors.New("Non comparable types")
	}
	thist := time.Unix(int64(t), 0)
	othert := time.Unix(int64(otherc), 0)
	diff := thist.Sub(othert)
	if time.Duration(math.Abs(float64(diff))) < 10*time.Second {
		return 0, nil
	}
	if diff > 0 {
		return 1, nil
	}
	return -1, nil
}

func ParseTimeIndication(input string) (data TimeIndication, err error) {
	var result time.Time
	for _, format := range []string{"15:04", "15:04:05", "2006-01-02 15:04", "2006-01-02 15:04:05"} {
		result, err = time.ParseInLocation(format, input, time.Local)
		if err == nil {
			if result.Year() == 0 {
				now := time.Now()
				result = time.Date(now.Year(), now.Month(), now.Day(), result.Hour(), result.Minute(), 59, int(1*time.Second-1*time.Nanosecond), result.Location())
				if result.Before(now) {
					result = result.Add(24 * time.Hour)
				}
			}
			data = TimeIndication(result.Unix())
			return
		}
	}

	return
}

func (sq *TimeIndication) UnmarshalJSON(data []byte) (err error) {
	var tmp string
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return
	}

	*sq, err = ParseTimeIndication(tmp)
	return
}

func (sq TimeIndication) MarshalJSON() ([]byte, error) {
	return []byte(time.Unix(int64(sq), 0).Format("\"2006-01-02 15:04:05\"")), nil
}

func StringToTimeIndication(sourceType, targetType reflect.Type, sourceData interface{}) (interface{}, error) {
	if sourceType.Kind() != reflect.String {
		return sourceData, nil
	}

	if targetType != reflect.TypeOf(TimeIndication(0)) {
		return sourceData, nil
	}

	return ParseTimeIndication(sourceData.(string))
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
	hooks = append(hooks, StringInterfaceToJsonRawMessage, StringToTimeIndication)
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
