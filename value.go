// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package expr

import (
	"strconv"

	"github.com/tidwall/gjson"
)

type value struct {
	raw  []byte
	path string
}

// MarshalJSON lets pure expression helpers consume a selected JSON value
// without exposing the internal path wrapper.
func (v value) MarshalJSON() ([]byte, error) {
	if v.path == "" {
		if len(v.raw) == 0 {
			return []byte("null"), nil
		}
		return v.raw, nil
	}
	result := gjson.GetBytes(v.raw, v.path)
	if !result.Exists() {
		return []byte("null"), nil
	}
	return []byte(result.Raw), nil
}

// Get returns a named child of the selected JSON value, or nil when it is absent.
func (v value) Get(name string) any {
	return v.lookup(name)
}

// Index returns an indexed child of the selected JSON value, or nil when it is absent.
func (v value) Index(index int) any {
	return v.lookup(strconv.Itoa(index))
}

func (v value) lookup(next string) any {
	path := next
	if v.path != "" {
		path = v.path + "." + next
	}
	return fromResult(gjson.GetBytes(v.raw, path), v.raw, path)
}

func fromResult(result gjson.Result, raw []byte, path string) any {
	if !result.Exists() {
		return nil
	}
	switch result.Type {
	case gjson.JSON:
		return resultValue(result)
	case gjson.String:
		return result.String()
	case gjson.Number:
		if i := result.Int(); strconv.FormatInt(i, 10) == result.Raw {
			return i
		}
		return result.Float()
	case gjson.True, gjson.False:
		return result.Bool()
	default:
		return nil
	}
}

func unwrap(v any) any {
	if wrapped, ok := v.(value); ok {
		if wrapped.path == "" {
			return decodeJSON(string(wrapped.raw))
		}
		return gjson.GetBytes(wrapped.raw, wrapped.path).Value()
	}
	return v
}

func decodeJSON(raw string) any {
	if !gjson.Valid(raw) {
		return nil
	}
	return resultValue(gjson.Parse(raw))
}

func resultValue(result gjson.Result) any {
	switch {
	case result.IsArray():
		count := 0
		result.ForEach(func(_, _ gjson.Result) bool { count++; return true })
		values := make([]any, 0, count)
		result.ForEach(func(_, item gjson.Result) bool {
			values = append(values, resultValue(item))
			return true
		})
		return values
	case result.IsObject():
		count := 0
		result.ForEach(func(_, _ gjson.Result) bool { count++; return true })
		values := make(map[string]any, count)
		result.ForEach(func(key, item gjson.Result) bool {
			values[key.String()] = resultValue(item)
			return true
		})
		return values
	}
	switch result.Type {
	case gjson.String:
		return result.String()
	case gjson.Number:
		if integer, err := strconv.ParseInt(result.Raw, 10, 64); err == nil {
			return integer
		}
		return result.Float()
	case gjson.True, gjson.False:
		return result.Bool()
	default:
		return nil
	}
}
