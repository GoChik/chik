package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/PaesslerAG/gval"
)

type State struct {
	Current  map[string]interface{} `json:"current"`
	Previous map[string]interface{} `json:"previous"`
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
	expression, err := gval.Full().NewEvaluable(key)
	if err != nil {
		return nil, err
	}

	expressionValue := func(data map[string]interface{}) (interface{}, error) {
		return expression(context.Background(), data)
	}

	currentValue, err := expressionValue(s.Current)
	if err != nil {
		return nil, err
	}
	previousValue, err := expressionValue(s.Previous)

	return &fieldDescriptor{
		currentValue,
		!reflect.DeepEqual(previousValue, currentValue),
	}, nil
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

type StateQueries []StateQuery

func queryFromRawData(raw map[string]interface{}) (StateQuery, error) {
	setValue := func(value interface{}) (StateQuery, error) {
		resultValue := reflect.ValueOf(value).Elem()
		for k, v := range raw {
			val := resultValue.FieldByName(strings.Title(k))
			if !val.IsValid() {
				return nil, fmt.Errorf("Cannot store key %s in %v", k, resultValue.Type())
			}
			val.Set(reflect.ValueOf(v))
		}
		return value.(StateQuery), nil
	}

	_, ok1 := raw["Const"]
	_, ok2 := raw["const"]
	if ok1 || ok2 {
		return setValue(&MixedQuery{})
	}
	return setValue(&StructQuery{})
}

func (sq *StateQueries) UnmarshalJSON(data []byte) error {
	temp := make([]interface{}, 0)
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	res := make(StateQueries, 0, len(temp))
	for _, q := range temp {
		value, err := queryFromRawData(q.(map[string]interface{}))
		if err != nil {
			return err
		}
		res = append(res, value)
	}
	*sq = res
	return nil
}

func StringInterfaceToStateQuery(sourceType, targetType reflect.Type, sourceData interface{}) (interface{}, error) {
	if sourceType.Kind() != reflect.Map {
		return sourceData, nil
	}

	if targetType != reflect.TypeOf((*StateQuery)(nil)).Elem() {
		return sourceData, nil
	}

	return queryFromRawData(sourceData.(map[string]interface{}))
}
