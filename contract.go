package expr

import (
	"encoding/json/jsontext"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Request is the shared expression evaluation request used by HTTP and Tools.
type Request struct {
	Expression string         `json:"expression"`
	Input      jsontext.Value `json:"input,omitempty"`
	At         string         `json:"at,omitempty"`
}

// Result is the stable typed result envelope returned by expression evaluation.
type Result struct {
	Type string `json:"type"`
	Out  any    `json:"out"`
}

// Failure is the structured output for a failed expression evaluation.
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Line    int    `json:"line,omitzero"`
	Column  int    `json:"column,omitzero"`
}

// Evaluate compiles and evaluates one request. Expression failures are returned
// in the typed result envelope; only request transport errors stay outside it.
func Evaluate(request Request) Result {
	if strings.TrimSpace(request.Expression) == "" {
		return failure("invalid_input", "expression is required", nil)
	}

	input, err := validateInput(request.Input)
	if err != nil {
		return failure(inputCode(err), err.Error(), err)
	}

	at := time.Now().UTC()
	if request.At != "" {
		at, err = time.Parse(time.RFC3339, request.At)
		if err != nil {
			return failure("invalid_input", fmt.Sprintf("at: %v", err), err)
		}
		at = at.UTC()
	}

	program, err := Compile(request.Expression)
	if err != nil {
		return failure(compileCode(err), err.Error(), err)
	}
	value, err := evalAt(program, input, nil, at)
	if err != nil {
		return failure(evalCode(err), err.Error(), err)
	}
	kind, out, err := result(value)
	if err != nil {
		return failure("evaluation_error", err.Error(), err)
	}
	return Result{Type: kind, Out: out}
}

func result(value any) (string, any, error) {
	if err := validateValue(value); err != nil {
		return "", nil, err
	}
	var kind string
	var out any
	switch x := value.(type) {
	case nil:
		kind = "null"
	case bool:
		kind, out = "boolean", x
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		kind, out = "integer", x
	case float32:
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return "", nil, fmt.Errorf("result contains a non-finite number")
		}
		kind, out = "number", x
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return "", nil, fmt.Errorf("result contains a non-finite number")
		}
		kind, out = "number", x
	case string:
		kind, out = "string", x
	case time.Time:
		kind, out = "time", x.UTC().Format(time.RFC3339Nano)
	case time.Duration:
		kind, out = "duration", x.String()
	case []any:
		kind, out = "array", x
	case map[string]any:
		kind, out = "object", x
	default:
		return "", nil, fmt.Errorf("result type %T is not representable", value)
	}
	if err := encodedResult(out); err != nil {
		return "", nil, err
	}
	return kind, out, nil
}

func encodedResult(value any) error {
	size, err := encodedSize(value, 1, new(int))
	if err != nil {
		return fmt.Errorf("result: %w", err)
	}
	if size > maxJSONBytes {
		return outputLimitError()
	}
	return nil
}

func encodedSize(value any, depth int, entries *int) (int, error) {
	if depth > maxNesting {
		return 0, fmt.Errorf("nesting exceeds %d", maxNesting)
	}
	switch value := value.(type) {
	case nil:
		return len("null"), nil
	case bool:
		if value {
			return len("true"), nil
		}
		return len("false"), nil
	case string:
		return encodedStringSize(value)
	case time.Time:
		return encodedStringSize(value.UTC().Format(time.RFC3339Nano))
	case time.Duration:
		return encodedStringSize(value.String())
	case int, int8, int16, int32, int64:
		var buf [32]byte
		return len(strconv.AppendInt(buf[:0], reflect.ValueOf(value).Int(), 10)), nil
	case uint, uint8, uint16, uint32, uint64:
		var buf [32]byte
		return len(strconv.AppendUint(buf[:0], reflect.ValueOf(value).Uint(), 10)), nil
	case float32:
		number := float64(value)
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return 0, fmt.Errorf("result contains a non-finite number")
		}
		var buf [32]byte
		return len(jsontext.AppendFloat(buf[:0], number, 32)), nil
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, fmt.Errorf("result contains a non-finite number")
		}
		var buf [32]byte
		return len(jsontext.AppendFloat(buf[:0], value, 64)), nil
	case []any:
		return encodedArraySize(value, depth, entries)
	case map[string]any:
		return encodedObjectSize(value, depth, entries)
	default:
		return 0, fmt.Errorf("type %T is not representable", value)
	}
}

func encodedArraySize(values []any, depth int, entries *int) (int, error) {
	size := 2
	for i, value := range values {
		if i > 0 {
			size++
		}
		item, err := encodedEntrySize(value, depth, entries)
		if err != nil {
			return 0, err
		}
		size += item
		if size > maxJSONBytes {
			return size, nil
		}
	}
	return size, nil
}

func encodedObjectSize(values map[string]any, depth int, entries *int) (int, error) {
	size, index := 2, 0
	for key, value := range values {
		if index > 0 {
			size++
		}
		keySize, err := encodedStringSize(key)
		if err != nil {
			return 0, err
		}
		item, err := encodedEntrySize(value, depth, entries)
		if err != nil {
			return 0, err
		}
		size += keySize + 1 + item
		if size > maxJSONBytes {
			return size, nil
		}
		index++
	}
	return size, nil
}

func encodedEntrySize(value any, depth int, entries *int) (int, error) {
	(*entries)++
	if *entries > maxEntries {
		return 0, fmt.Errorf("collection entries exceed %d", maxEntries)
	}
	return encodedSize(value, depth+1, entries)
}

func encodedStringSize(value string) (int, error) {
	if !utf8.ValidString(value) {
		return 0, fmt.Errorf("string contains invalid utf-8")
	}
	size := 2
	for _, r := range value {
		if r < 0x20 {
			switch r {
			case '\b', '\f', '\n', '\r', '\t':
				size += 2
			default:
				size += 6
			}
			continue
		}
		switch r {
		case '"', '\\':
			size += 2
		default:
			size += utf8.RuneLen(r)
		}
	}
	return size, nil
}

func outputJSONValue(value any) any {
	if !needsOutputConversion(value) {
		return value
	}
	return convertOutputValue(value)
}

func needsOutputConversion(value any) bool {
	switch x := value.(type) {
	case time.Time, time.Duration:
		return true
	case []any:
		for _, item := range x {
			if needsOutputConversion(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range x {
			if needsOutputConversion(item) {
				return true
			}
		}
	}
	return false
}

func convertOutputValue(value any) any {
	switch x := value.(type) {
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case time.Duration:
		return x.String()
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = convertOutputValue(x[i])
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for key, item := range x {
			out[key] = convertOutputValue(item)
		}
		return out
	default:
		return value
	}
}

func failure(code, message string, cause error) Result {
	out := Failure{Code: code, Message: message}
	if cause != nil {
		out.Line, out.Column, _ = location(cause)
	}
	return Result{Type: "error", Out: out}
}

func inputCode(err error) string {
	if strings.Contains(err.Error(), "exceeds") {
		return "limit_exceeded"
	}
	return "invalid_input"
}

func compileCode(err error) string {
	if strings.Contains(err.Error(), "exceeds") {
		return "limit_exceeded"
	}
	return "compile_error"
}

func evalCode(err error) string {
	message := err.Error()
	if strings.Contains(message, "memory budget") || strings.Contains(message, "nesting exceeds") || strings.Contains(message, "collection entries exceed") || strings.Contains(message, "output exceeds") {
		return "limit_exceeded"
	}
	return "evaluation_error"
}
