package actor

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
)

type State struct {
	Current  interface{}
	Previous interface{}
}

func (s *State) GetField(key string) (interface{}, error) {
	slices := strings.Split(key, ".")
	logrus.Debug(slices, s)
	value := reflect.ValueOf(s).Elem()
	for _, slice := range slices {
		var tmp reflect.Value
		switch value.Kind() {
		case reflect.Map:
			tmp = value.MapIndex(reflect.ValueOf(slice))
			if !tmp.IsValid() {
				tmp = value.MapIndex(reflect.ValueOf(strings.ToLower(slice)))
			}

		case reflect.Struct:
			tmp = value.FieldByName(slice)
			if !tmp.IsValid() {
				tmp = value.FieldByName(strings.ToLower(slice))
			}
		}

		if !tmp.IsValid() {
			return nil, fmt.Errorf("Cannot find field with name %s in %+v", slice, s)
		}
		if tmp.Kind() == reflect.Interface || tmp.Kind() == reflect.Ptr {
			value = tmp.Elem()
		} else {
			value = tmp
		}
	}
	return value.Interface(), nil
}

type StateQuery interface {
	Execute(state *State) (bool, error)
}

// StructQuery compares two elements of the State oin every state change
type StructQuery struct {
	FirstField  string
	Operator    string
	SecondField string
}

func (q *StructQuery) Execute(state *State) (bool, error) {
	firstValue, err := state.GetField(q.FirstField)
	if err != nil {
		return false, err
	}
	secondValue, err := state.GetField(q.SecondField)
	if err != nil {
		return false, err
	}

	return Compare(firstValue, secondValue, q.Operator)
}

// MixedQuery compares an element of State.Current with a constant only if that element is different from the same in State.Previous
type MixedQuery struct {
	Field    string
	Operator string
	Constant interface{}
}

func (q *MixedQuery) Execute(state *State) (bool, error) {
	slices := strings.Split(q.Field, ".")
	if len(slices) == 0 {
		return false, errors.New("Cannot execute query with null field")
	}

	for _, target := range []string{"state", "current", "previous"} {
		if strings.ToLower(slices[0]) == target {
			slices = slices[1:]
		}
	}

	queryValue := func(root string) string {
		return strings.Join(append([]string{root}, slices...), ".")
	}

	currentValue, err := state.GetField(queryValue("current"))
	if err != nil {
		return false, err
	}
	previousValue, err := state.GetField(queryValue("previous"))

	if reflect.DeepEqual(previousValue, currentValue) {
		return false, nil
	}

	return Compare(currentValue, q.Constant, q.Operator)
}

func StringInterfaceToStateQuery(sourceType, targetType reflect.Type, sourceData interface{}) (interface{}, error) {
	if sourceType.Kind() != reflect.Map {
		return sourceData, nil
	}

	if targetType != reflect.TypeOf((*StateQuery)(nil)).Elem() {
		return sourceData, nil
	}

	mapSource := sourceData.(map[string]interface{})
	setValue := func(value interface{}) (interface{}, error) {
		resultValue := reflect.ValueOf(value).Elem()
		for k, v := range mapSource {
			val := resultValue.FieldByName(strings.Title(k))
			if !val.IsValid() {
				return nil, fmt.Errorf("Cannot store key %s in %v", k, resultValue.Type())
			}
			val.Set(reflect.ValueOf(v))
		}
		return value, nil
	}

	_, ok1 := mapSource["Constant"]
	_, ok2 := mapSource["constant"]
	if ok1 || ok2 {
		return setValue(&MixedQuery{})
	}
	return setValue(&StructQuery{})
}
