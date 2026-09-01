package expr

import (
	"encoding/json/jsontext"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/tidwall/gjson"
)

var emptyObject = []byte("{}")

func validateInput(raw []byte) ([]byte, error) {
	switch {
	case len(raw) == 0:
		return emptyObject, nil
	case len(raw) > maxJSONBytes:
		return nil, fmt.Errorf("expression: input exceeds %d bytes", maxJSONBytes)
	}
	if err := validateJSON(raw); err != nil {
		return nil, fmt.Errorf("expression: input: %w", err)
	}
	return raw, nil
}

func validateJSON(raw []byte) error {
	if !jsontext.Value(raw).IsValid() {
		return fmt.Errorf("must be valid json")
	}
	text := unsafe.String(unsafe.SliceData(raw), len(raw))
	return validateJSONResult(gjson.Parse(text), 1, new(int))
}

func validateJSONResult(result gjson.Result, depth int, entries *int) error {
	switch {
	case depth > maxNesting:
		return fmt.Errorf("expression: nesting exceeds %d", maxNesting)
	case result.Type == gjson.Number && (math.IsNaN(result.Float()) || math.IsInf(result.Float(), 0)):
		return fmt.Errorf("expression: result contains a non-finite number")
	case result.Type != gjson.JSON:
		return nil
	}
	var walkErr error
	result.ForEach(func(_, item gjson.Result) bool {
		(*entries)++
		if *entries > maxEntries {
			walkErr = fmt.Errorf("expression: collection entries exceed %d", maxEntries)
			return false
		}
		walkErr = validateJSONResult(item, depth+1, entries)
		return walkErr == nil
	})
	return walkErr
}

func validateOutput(raw []byte) error {
	if len(raw) > maxJSONBytes {
		return outputLimitError()
	}
	if err := validateJSON(raw); err != nil {
		return fmt.Errorf("expression: output: %w", err)
	}
	return nil
}

// ValidateOutput validates one JSON result against the expression output
// contract, including the 256 KiB size and structural limits.
func ValidateOutput(raw []byte) error { return validateOutput(raw) }

func outputLimitError() error {
	return fmt.Errorf("expression: output exceeds %d bytes", maxJSONBytes)
}

func validateValue(v any) error {
	_, err := normalizeValue(v, 1, new(int))
	return err
}

func normalizeValue(v any, depth int, entries *int) (any, error) {
	if depth > maxNesting {
		return nil, fmt.Errorf("expression: nesting exceeds %d", maxNesting)
	}
	switch x := v.(type) {
	case nil, bool, string:
		return x, nil
	case time.Time:
		return x.UTC(), nil
	case time.Duration:
		return x, nil
	case jsontext.Value:
		return normalizeJSONNumber(x)
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(x).Int(), nil
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(x).Uint(), nil
	case float32, float64:
		return normalizeFloat(reflect.ValueOf(x).Float())
	case []any:
		return normalizeSlice(x, depth, entries)
	case map[string]any:
		return normalizeMap(x, depth, entries)
	}
	return normalizeReflected(v, depth, entries)
}

func normalizeJSONNumber(value jsontext.Value) (any, error) {
	if integer, err := strconv.ParseInt(string(value), 10, 64); err == nil {
		return integer, nil
	}
	float, err := strconv.ParseFloat(string(value), 64)
	if err != nil {
		return nil, fmt.Errorf("expression: result contains a non-finite number")
	}
	return normalizeFloat(float)
}

func normalizeFloat(value float64) (float64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("expression: result contains a non-finite number")
	}
	return value, nil
}

func normalizeReflected(v any, depth int, entries *int) (any, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		return normalizeReflectedSlice(rv, v, depth, entries)
	case reflect.Map:
		return normalizeReflectedMap(rv, depth, entries)
	default:
		return nil, fmt.Errorf("expression: result type %T is not representable", v)
	}
}

func normalizeReflectedSlice(rv reflect.Value, original any, depth int, entries *int) ([]any, error) {
	if rv.Type().Elem().Kind() == reflect.Uint8 {
		return nil, fmt.Errorf("expression: result type %T is not representable", original)
	}
	out := make([]any, rv.Len())
	for i := range out {
		(*entries)++
		if *entries > maxEntries {
			return nil, fmt.Errorf("expression: collection entries exceed %d", maxEntries)
		}
		item, err := normalizeValue(rv.Index(i).Interface(), depth+1, entries)
		if err != nil {
			return nil, err
		}
		out[i] = item
	}
	return out, nil
}

func normalizeReflectedMap(rv reflect.Value, depth int, entries *int) (map[string]any, error) {
	if rv.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("expression: result map keys must be strings")
	}
	out := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		(*entries)++
		if *entries > maxEntries {
			return nil, fmt.Errorf("expression: collection entries exceed %d", maxEntries)
		}
		item, err := normalizeValue(iter.Value().Interface(), depth+1, entries)
		if err != nil {
			return nil, err
		}
		out[iter.Key().String()] = item
	}
	return out, nil
}

func normalizeSlice(values []any, depth int, entries *int) ([]any, error) {
	for i, value := range values {
		(*entries)++
		if *entries > maxEntries {
			return nil, fmt.Errorf("expression: collection entries exceed %d", maxEntries)
		}
		item, err := normalizeValue(value, depth+1, entries)
		if err != nil {
			return nil, err
		}
		values[i] = item
	}
	return values, nil
}

func normalizeMap(values map[string]any, depth int, entries *int) (map[string]any, error) {
	for key, value := range values {
		(*entries)++
		if *entries > maxEntries {
			return nil, fmt.Errorf("expression: collection entries exceed %d", maxEntries)
		}
		item, err := normalizeValue(value, depth+1, entries)
		if err != nil {
			return nil, err
		}
		values[key] = item
	}
	return values, nil
}
