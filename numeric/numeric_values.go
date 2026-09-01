// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package numeric

import (
	"fmt"
	"math"
	"reflect"
)

func numbers(value any) ([]float64, error) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || (rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice) {
		return nil, fmt.Errorf("numeric array is required")
	}
	out := make([]float64, rv.Len())
	for i := range out {
		value, err := number(rv.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("numeric array index %d: %w", i, err)
		}
		out[i] = value
	}
	return out, nil
}

func number(value any) (float64, error) {
	switch x := value.(type) {
	case int:
		return float64(x), nil
	case int8:
		return float64(x), nil
	case int16:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case uint:
		return float64(x), nil
	case uint8:
		return float64(x), nil
	case uint16:
		return float64(x), nil
	case uint32:
		return float64(x), nil
	case uint64:
		return float64(x), nil
	case float32:
		return finiteNumber(float64(x))
	case float64:
		return finiteNumber(x)
	default:
		return 0, fmt.Errorf("value must be a finite number")
	}
}

func finiteNumber(value float64) (float64, error) {
	if !finite(value) {
		return 0, fmt.Errorf("value must be a finite number")
	}
	return value, nil
}

func integer(value any) (int, error) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	switch x := value.(type) {
	case int:
		return x, nil
	case int8:
		return int(x), nil
	case int16:
		return int(x), nil
	case int32:
		return int(x), nil
	case int64:
		if x < int64(minInt) || x > int64(maxInt) {
			return 0, fmt.Errorf("value must be an integer")
		}
		return int(x), nil
	case uint:
		if uint64(x) > uint64(maxInt) {
			return 0, fmt.Errorf("value must be an integer")
		}
		return int(x), nil
	case uint8:
		return int(x), nil
	case uint16:
		return int(x), nil
	case uint32:
		if uint64(x) > uint64(maxInt) {
			return 0, fmt.Errorf("value must be an integer")
		}
		return int(x), nil
	case uint64:
		if x > uint64(maxInt) {
			return 0, fmt.Errorf("value must be an integer")
		}
		return int(x), nil
	case float32:
		return integerFloat(float64(x), minInt, maxInt)
	case float64:
		return integerFloat(x, minInt, maxInt)
	default:
		return 0, fmt.Errorf("value must be an integer")
	}
}

func integerFloat(value float64, minInt, maxInt int) (int, error) {
	if !finite(value) || value != math.Trunc(value) || value < float64(minInt) || value >= float64(maxInt)+1 {
		return 0, fmt.Errorf("value must be an integer")
	}
	return int(value), nil
}

func oneNumber(args []any, name string) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("%s: expected one argument", name)
	}
	return number(args[0])
}

func text(value any) (string, error) {
	out, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value must be a string")
	}
	return out, nil
}

func pair(x, y []float64, name string) error {
	if len(x) == 0 || len(y) == 0 {
		return fmt.Errorf("%s: values must not be empty", name)
	}
	if len(x) != len(y) {
		return fmt.Errorf("%s: arrays must have equal lengths", name)
	}
	if err := finiteValues(x); err != nil {
		return err
	}
	return finiteValues(y)
}

func finiteValues(values []float64) error {
	for i, value := range values {
		if !finite(value) {
			return fmt.Errorf("numeric array index %d: value must be finite", i)
		}
	}
	return nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
