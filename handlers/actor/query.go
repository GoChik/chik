package actor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/gochik/chik/types"
)

type State struct {
	Current  map[string]interface{} `json:"current"`
	Previous map[string]interface{} `json:"previous"`
	language gval.Language
}

func CreateState(previous, current map[string]interface{}) *State {
	return &State{
		Previous: previous,
		Current:  current,
		language: gval.Full(
			gval.Function("time", func(args ...interface{}) (interface{}, error) {
				strdate, ok := args[0].(string)
				if !ok {
					intdate, ok := args[0].(int64)
					if ok {
						return types.TimeIndication(intdate), nil
					}
					return nil, errors.New("Wrong argument given to function duration")
				}
				return types.ParseTimeIndication(strdate)
			}),

			gval.Function("duration", func(args ...interface{}) (interface{}, error) {
				strdate, ok := args[0].(string)
				if !ok {
					return nil, errors.New("Wrong argument given to function duration")
				}
				return time.ParseDuration(strdate)
			}),

			gval.InfixOperator("after", func(a, b interface{}) (interface{}, error) {
				date, ok1 := a.(types.TimeIndication)
				duration, ok2 := b.(time.Duration)
				if ok1 && ok2 {
					return types.TimeIndication(time.Unix(int64(date), 0).Add(duration).Unix()), nil
				}
				return nil, fmt.Errorf("Failed to execute + on non time arguments: %T %T %v %v", a, duration, ok1, ok2)
			}),
		),
	}
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
	expression, err := s.language.NewEvaluable(key)

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
	Var1      string `json:"var1"`
	Op        string `json:"op"`
	Var2      string `json:"var2"`
	Detrigger bool   `json:"disable_trigger_on_change,omitempty"`
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
		"Query between %v:%T and %v:%T result: %v, %v",
		firstValue.value, firstValue.value,
		secondValue.value, secondValue.value,
		match,
		firstValue.changedSincePreviousUpdate || secondValue.changedSincePreviousUpdate,
	)

	res = QueryResult{
		match:                          match,
		changedSincePreviousEvaluation: !q.Detrigger && (firstValue.changedSincePreviousUpdate || secondValue.changedSincePreviousUpdate),
	}

	return
}

// MixedQuery compares an element of State.Current with a constant only if that element is different from the same in State.Previous
type MixedQuery struct {
	Var1      string      `json:"var1"`
	Op        string      `json:"op"`
	Const     interface{} `json:"const"`
	Detrigger bool        `json:"disable_trigger_on_change,omitempty"`
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
		Msgf("Query between %v:%T and %v result: %v, %v", currentValue.value, currentValue.value, q.Const, match, currentValue.changedSincePreviousUpdate)

	res = QueryResult{
		match:                          match,
		changedSincePreviousEvaluation: !q.Detrigger && currentValue.changedSincePreviousUpdate,
	}

	return
}

type StateQueries []StateQuery

func queryFromRawData(raw map[string]interface{}) (StateQuery, error) {
	setValue := func(value interface{}) (StateQuery, error) {
		resultValue := reflect.ValueOf(value).Elem()
		for k, v := range raw {
			val, ok := resultValue.Type().FieldByNameFunc(func(fieldName string) bool {
				field, _ := resultValue.Type().FieldByName(fieldName)
				tagName := strings.Split(field.Tag.Get("json"), ",")[0]
				switch tagName {
				case k:
					return true
				case "":
					if fieldName == k {
						return true
					}
				}
				return false
			})
			if !ok {
				return nil, fmt.Errorf("Cannot store key %s in %v", k, resultValue)
			}
			resultValue.FieldByIndex(val.Index).Set(reflect.ValueOf(v))
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
