package actor

import (
	"fmt"
	"reflect"
	"strings"
)

type State struct {
	Current  interface{} `json:"current"`
	Previous interface{} `json:"previous"`
}

type QueryResult struct {
	match                          bool
	changedSincePreviousEvaluation bool
}

type fieldDescriptor struct {
	value                      interface{}
	changedSincePreviousUpdate bool
}

func valueMatch(value reflect.StructField, name string) bool {
	if value.Name == name {
		return true
	}

	if strings.Split(value.Tag.Get("json"), ",")[0] == name {
		return true
	}

	return false
}

func (s *State) GetFieldDescriptor(key string) (*fieldDescriptor, error) {
	slices := strings.Split(key, ".")
	queryValue := func(root string) []string {
		return append([]string{root}, slices...)
	}

	currentValue, err := s.GetField(queryValue("current"))
	if err != nil {
		return nil, err
	}
	previousValue, err := s.GetField(queryValue("previous"))

	return &fieldDescriptor{
		currentValue,
		!reflect.DeepEqual(previousValue, currentValue),
	}, nil
}

func (s *State) GetField(slices []string) (interface{}, error) {
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
			for i := 0; i < value.NumField(); i++ {
				if valueMatch(value.Type().Field(i), slice) {
					tmp = value.Field(i)
					break
				}
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
	Execute(state *State) (QueryResult, error)
}

// StructQuery compares two elements of the State oin every state change
type StructQuery struct {
	Var1 string `json:"var1"`
	Op   string `json:"op"`
	Var2 string `json:"var2"`
}

func (q *StructQuery) Execute(state *State) (res QueryResult, err error) {
	logger.Info().Str("query_type", "struct").Msgf("Executing query: %v %v %v", q.Var1, q.Op, q.Var2)
	firstValue, err := state.GetFieldDescriptor(q.Var1)
	if err != nil {
		return
	}
	secondValue, err := state.GetFieldDescriptor(q.Var2)
	if err != nil {
		return
	}

	match, err := Compare(firstValue.value, secondValue.value, q.Op)
	if err != nil {
		return
	}

	logger.Debug().Str("query_type", "struct").Msgf(
		"Query between %v and %v result: %v, %v",
		firstValue.value,
		secondValue.value,
		match,
		firstValue.changedSincePreviousUpdate || secondValue.changedSincePreviousUpdate,
	)

	res = QueryResult{
		match:                          match,
		changedSincePreviousEvaluation: firstValue.changedSincePreviousUpdate || secondValue.changedSincePreviousUpdate,
	}

	return
}

// MixedQuery compares an element of State.Current with a constant only if that element is different from the same in State.Previous
type MixedQuery struct {
	Var1  string      `json:"var1"`
	Op    string      `json:"op"`
	Const interface{} `json:"const"`
}

func (q *MixedQuery) Execute(state *State) (res QueryResult, err error) {
	logger.Info().Str("query_type", "mixed").Msgf("Executing mixed query: %v %v %v", q.Var1, q.Op, q.Const)
	currentValue, err := state.GetFieldDescriptor(q.Var1)
	if err != nil {
		return
	}

	match, err := Compare(currentValue.value, q.Const, q.Op)
	if err != nil {
		return
	}

	logger.Debug().
		Str("query_type", "mixed").
		Msgf("Query between %v and %v result: %v, %v", currentValue.value, q.Const, match, currentValue.changedSincePreviousUpdate)

	res = QueryResult{
		match:                          match,
		changedSincePreviousEvaluation: currentValue.changedSincePreviousUpdate,
	}

	return
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

	_, ok1 := mapSource["Const"]
	_, ok2 := mapSource["const"]
	if ok1 || ok2 {
		return setValue(&MixedQuery{})
	}
	return setValue(&StructQuery{})
}
