// Package assignment provides deterministic experiment assignment helpers.
package assignment

import (
	"encoding/json/v2"
	"fmt"

	exprlib "github.com/expr-lang/expr"
	"github.com/kelindar/expr/encoding"
	"github.com/zeebo/xxh3"
)

// Bucket returns a deterministic zero-based bucket for a JSON value.
func Bucket(value any, count int, namespace ...string) (int, error) {
	switch {
	case count <= 0:
		return 0, fmt.Errorf("bucket: count must be positive")
	case len(namespace) > 1:
		return 0, fmt.Errorf("bucket: expected zero or one namespace")
	}
	var input any = value
	if len(namespace) == 1 {
		input = []any{namespace[0], value}
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return 0, fmt.Errorf("bucket: value is not representable as json: %w", err)
	}
	// json.Marshal always produces valid JSON, so canonicalization cannot fail here.
	canonical, _ := encoding.CanonicalJSON(raw)
	return int(xxh3.Hash(canonical) % uint64(count)), nil
}

// Options returns the Expr registrations for deterministic assignment helpers.
func Options() []exprlib.Option {
	return []exprlib.Option{
		exprlib.Function("bucket", bucket, new(func(...any) any)),
	}
}

func bucket(args ...any) (any, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("bucket: expected two or three arguments")
	}
	count, err := integer(args[1])
	if err != nil {
		return nil, err
	}
	if len(args) == 3 {
		namespace, ok := args[2].(string)
		if !ok {
			return nil, fmt.Errorf("bucket: namespace must be a string")
		}
		return Bucket(args[0], count, namespace)
	}
	return Bucket(args[0], count)
}

func integer(value any) (int, error) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	var signed int64
	var unsigned uint64
	var isUnsigned bool
	switch x := value.(type) {
	case int:
		return x, nil
	case int8:
		signed = int64(x)
	case int16:
		signed = int64(x)
	case int32:
		signed = int64(x)
	case int64:
		signed = x
	case uint:
		unsigned, isUnsigned = uint64(x), true
	case uint8:
		unsigned, isUnsigned = uint64(x), true
	case uint16:
		unsigned, isUnsigned = uint64(x), true
	case uint32:
		unsigned, isUnsigned = uint64(x), true
	case uint64:
		unsigned, isUnsigned = x, true
	default:
		return 0, fmt.Errorf("bucket: count must be an integer")
	}
	if isUnsigned {
		if unsigned > uint64(maxInt) {
			return 0, fmt.Errorf("bucket: count must be an integer")
		}
		return int(unsigned), nil
	}
	if signed < int64(minInt) || signed > int64(maxInt) {
		return 0, fmt.Errorf("bucket: count must be an integer")
	}
	return int(signed), nil
}
