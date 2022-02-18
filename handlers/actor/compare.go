package actor

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gochik/chik/types"
)

const (
	greaterOp  = ">"
	lessOp     = "<"
	equalOp    = "=="
	notEqualOp = "!="
)

func equal(x, y interface{}) (bool, error) {
	cx, ok1 := x.(types.Comparable)
	cy, ok2 := y.(types.Comparable)
	if ok1 && ok2 {
		res, err := cx.Compare(cy)
		return res == 0, err
	}

	return reflect.DeepEqual(x, y), nil
}

func greater(x, y interface{}) (bool, error) {
	cx, ok1 := x.(types.Comparable)
	cy, ok2 := y.(types.Comparable)
	if ok1 && ok2 {
		res, err := cx.Compare(cy)
		return res == 1, err
	}

	var val1 float64
	var val2 float64
	var ok bool
	if val1, ok = x.(float64); !ok {
		return false, errors.New("Cannot convert operand to numeric value")
	}

	if val2, ok = y.(float64); !ok {
		return false, errors.New("Cannot convert operand to numeric value")
	}

	return val1 > val2, nil
}

func less(x, y interface{}) (bool, error) {
	cx, ok1 := x.(types.Comparable)
	cy, ok2 := y.(types.Comparable)
	if ok1 && ok2 {
		res, err := cx.Compare(cy)
		return res == -1, err
	}

	var val1 float64
	var val2 float64
	var ok bool
	if val1, ok = x.(float64); !ok {
		return false, errors.New("Cannot convert operand to numeric value")
	}

	if val2, ok = y.(float64); !ok {
		return false, errors.New("Cannot convert operand to numeric value")
	}

	return val1 < val2, nil
}

func Compare(x, y interface{}, operator string) (bool, error) {
	switch strings.Trim(operator, " ") {
	case equalOp:
		return equal(x, y)

	case notEqualOp:
		val, err := equal(x, y)
		return !val, err

	case greaterOp:
		return greater(x, y)

	case lessOp:
		return less(x, y)
	}

	return false, fmt.Errorf("Unknown operator: %s", operator)
}
