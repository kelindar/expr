// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

// Package collection provides bounded, deterministic collection operations.
package collection

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"reflect"
	"strconv"

	exprlib "github.com/expr-lang/expr"
)

// Chunk splits values into ordered, non-empty chunks of at most size elements.
func Chunk(values []any, size int) ([][]any, error) {
	if size <= 0 {
		return nil, fmt.Errorf("chunk: size must be positive")
	}
	capacity := len(values) / size
	if len(values)%size != 0 {
		capacity++
	}
	out := make([][]any, 0, capacity)
	for start := 0; start < len(values); {
		end := len(values)
		if size < len(values)-start {
			end = start + size
		}
		out = append(out, append([]any(nil), values[start:end]...))
		start = end
	}
	return out, nil
}

// Zip pairs values at matching positions and requires equal-length inputs.
func Zip(left, right []any) ([][]any, error) {
	if len(left) != len(right) {
		return nil, fmt.Errorf("zip: inputs must have equal lengths")
	}
	out := make([][]any, len(left))
	for i := range left {
		out[i] = []any{left[i], right[i]}
	}
	return out, nil
}

// Merge copies two objects, with keys from right replacing keys from left.
func Merge(left, right map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(left)+len(right))
	for key, value := range left {
		out[key] = value
	}
	for key, value := range right {
		out[key] = value
	}
	return out, nil
}

// Union returns unique values from left followed by unique values from right.
func Union(left, right []any) ([]any, error) { return set(left, right, "union") }

// Intersection returns unique values present in both inputs, preserving left order.
func Intersection(left, right []any) ([]any, error) {
	rightSet, err := keys(right)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	out := make([]any, 0, len(left))
	for _, value := range left {
		key, err := key(value)
		if err != nil {
			return nil, err
		}
		if _, ok := rightSet[key]; !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

// Difference returns unique values from left that are absent from right.
func Difference(left, right []any) ([]any, error) {
	rightSet, err := keys(right)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	out := make([]any, 0, len(left))
	for _, value := range left {
		key, err := key(value)
		if err != nil {
			return nil, err
		}
		if _, ok := rightSet[key]; ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

// Lag shifts values by periods and fills the leading positions with nil.
func Lag(values []any, periods int) ([]any, error) {
	if periods < 0 {
		return nil, fmt.Errorf("lag: periods must be non-negative")
	}
	out := make([]any, len(values))
	for i := range out {
		if i >= periods {
			out[i] = values[i-periods]
		}
	}
	return out, nil
}

// Cumsum returns the finite running sum of numeric values.
func Cumsum(values []any) ([]float64, error) {
	out := make([]float64, len(values))
	var total float64
	for i, value := range values {
		n, err := number(value)
		if err != nil {
			return nil, fmt.Errorf("cumsum index %d: %w", i, err)
		}
		total += n
		if !finite(total) {
			return nil, fmt.Errorf("cumsum: result is not finite")
		}
		out[i] = total
	}
	return out, nil
}

// Diff returns finite adjacent differences of numeric values.
func Diff(values []any) ([]float64, error) {
	if len(values) < 2 {
		return []float64{}, nil
	}
	out := make([]float64, len(values)-1)
	previous, err := number(values[0])
	if err != nil {
		return nil, fmt.Errorf("diff index 0: %w", err)
	}
	for i := 1; i < len(values); i++ {
		current, err := number(values[i])
		if err != nil {
			return nil, fmt.Errorf("diff index %d: %w", i, err)
		}
		out[i-1] = current - previous
		if !finite(out[i-1]) {
			return nil, fmt.Errorf("diff: result is not finite")
		}
		previous = current
	}
	return out, nil
}

// Options returns the Expr registrations for collection helpers.
func Options() []exprlib.Option {
	return []exprlib.Option{
		fn("chunk", chunkFunc),
		fn("zip", zipFunc),
		fn("merge", mergeFunc),
		fn("union", unionFunc),
		fn("intersection", intersectionFunc),
		fn("difference", differenceFunc),
		fn("lag", lagFunc),
		fn("cumsum", cumsumFunc),
		fn("diff", diffFunc),
	}
}

func chunkFunc(args ...any) (any, error) {
	if err := arity(args, 2, "chunk"); err != nil {
		return nil, err
	}
	values, err := array(args[0])
	if err != nil {
		return nil, err
	}
	size, err := integer(args[1])
	if err != nil {
		return nil, err
	}
	return Chunk(values, size)
}

func zipFunc(args ...any) (any, error) {
	left, right, err := pairArrays(args, "zip")
	if err != nil {
		return nil, err
	}
	return Zip(left, right)
}

func mergeFunc(args ...any) (any, error) {
	if err := arity(args, 2, "merge"); err != nil {
		return nil, err
	}
	left, err := object(args[0])
	if err != nil {
		return nil, err
	}
	right, err := object(args[1])
	if err != nil {
		return nil, err
	}
	return Merge(left, right)
}

func unionFunc(args ...any) (any, error) {
	left, right, err := pairArrays(args, "union")
	if err != nil {
		return nil, err
	}
	return Union(left, right)
}

func intersectionFunc(args ...any) (any, error) {
	left, right, err := pairArrays(args, "intersection")
	if err != nil {
		return nil, err
	}
	return Intersection(left, right)
}

func differenceFunc(args ...any) (any, error) {
	left, right, err := pairArrays(args, "difference")
	if err != nil {
		return nil, err
	}
	return Difference(left, right)
}

func lagFunc(args ...any) (any, error) {
	if err := arity(args, 2, "lag"); err != nil {
		return nil, err
	}
	values, err := array(args[0])
	if err != nil {
		return nil, err
	}
	periods, err := integer(args[1])
	if err != nil {
		return nil, err
	}
	return Lag(values, periods)
}

func cumsumFunc(args ...any) (any, error) {
	if err := arity(args, 1, "cumsum"); err != nil {
		return nil, err
	}
	values, err := array(args[0])
	if err != nil {
		return nil, err
	}
	return Cumsum(values)
}

func diffFunc(args ...any) (any, error) {
	if err := arity(args, 1, "diff"); err != nil {
		return nil, err
	}
	values, err := array(args[0])
	if err != nil {
		return nil, err
	}
	return Diff(values)
}

func arity(args []any, want int, name string) error {
	if len(args) != want {
		return fmt.Errorf("%s: expected %d arguments", name, want)
	}
	return nil
}

func pairArrays(args []any, name string) ([]any, []any, error) {
	if err := arity(args, 2, name); err != nil {
		return nil, nil, err
	}
	left, err := array(args[0])
	if err != nil {
		return nil, nil, err
	}
	right, err := array(args[1])
	if err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

func fn(name string, call func(...any) (any, error)) exprlib.Option {
	return exprlib.Function(name, call, new(func(...any) any))
}

func array(value any) ([]any, error) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || (rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice) {
		return nil, fmt.Errorf("array is required")
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out, nil
}

func object(value any) (map[string]any, error) {
	out, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("object is required")
	}
	return out, nil
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

func number(value any) (float64, error) {
	var out float64
	switch x := value.(type) {
	case int:
		out = float64(x)
	case int8:
		out = float64(x)
	case int16:
		out = float64(x)
	case int32:
		out = float64(x)
	case int64:
		out = float64(x)
	case uint:
		out = float64(x)
	case uint8:
		out = float64(x)
	case uint16:
		out = float64(x)
	case uint32:
		out = float64(x)
	case uint64:
		out = float64(x)
	case float32:
		out = float64(x)
	case float64:
		out = x
	default:
		return 0, fmt.Errorf("value must be a finite number")
	}
	if !finite(out) {
		return 0, fmt.Errorf("value must be a finite number")
	}
	return out, nil
}

func keys(values []any) (map[string]struct{}, error) {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		key, err := key(value)
		if err != nil {
			return nil, err
		}
		out[key] = struct{}{}
	}
	return out, nil
}

func key(value any) (string, error) {
	if value == nil {
		return "null", nil
	}
	raw, err := json.Marshal(comparable(value), json.Deterministic(true))
	if err != nil {
		return "", fmt.Errorf("collection value is not representable as json: %w", err)
	}
	return string(raw), nil
}

// comparable normalizes numeric representations so Expr's integer and float
// values compare alike in set operations while preserving nested structure.
func comparable(value any) any {
	switch x := value.(type) {
	case int:
		return jsontext.Value(strconv.FormatInt(int64(x), 10))
	case int8:
		return jsontext.Value(strconv.FormatInt(int64(x), 10))
	case int16:
		return jsontext.Value(strconv.FormatInt(int64(x), 10))
	case int32:
		return jsontext.Value(strconv.FormatInt(int64(x), 10))
	case int64:
		return jsontext.Value(strconv.FormatInt(x, 10))
	case uint:
		return jsontext.Value(strconv.FormatUint(uint64(x), 10))
	case uint8:
		return jsontext.Value(strconv.FormatUint(uint64(x), 10))
	case uint16:
		return jsontext.Value(strconv.FormatUint(uint64(x), 10))
	case uint32:
		return jsontext.Value(strconv.FormatUint(uint64(x), 10))
	case uint64:
		return jsontext.Value(strconv.FormatUint(x, 10))
	case float32:
		return jsontext.Value(strconv.FormatFloat(float64(x), 'g', -1, 32))
	case float64:
		return jsontext.Value(strconv.FormatFloat(x, 'g', -1, 64))
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = comparable(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for key, item := range x {
			out[key] = comparable(item)
		}
		return out
	default:
		return value
	}
}

func set(left, right []any, name string) ([]any, error) {
	seen := make(map[string]struct{}, len(left)+len(right))
	out := make([]any, 0, len(left)+len(right))
	for _, values := range [][]any{left, right} {
		for _, value := range values {
			key, err := key(value)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, value)
		}
	}
	return out, nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
