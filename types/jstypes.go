package types

import (
	"encoding/json"
	"reflect"

	"github.com/cloudflare/cfssl/log"
	"github.com/mitchellh/mapstructure"
)

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
