package types

import (
	"encoding/json"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// TimeIndication defines a basic struct for time with resolution of 1 min
type TimeIndication struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
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
	hooks = append(hooks, StringInterfaceToJsonRawMessage)
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
