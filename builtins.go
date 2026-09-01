// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package expr

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tidwall/gjson"
)

func getFunc(root value) func(string) any {
	return func(path string) any {
		return fromResult(gjson.GetBytes(root.raw, path), root.raw, path)
	}
}

func jsonFunc(root value) func(...any) (any, error) {
	return func(args ...any) (any, error) {
		switch len(args) {
		case 1:
			path, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("json: path must be a string")
			}
			return fromResult(gjson.GetBytes(root.raw, path), root.raw, path), nil
		case 2:
			path, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("json: path must be a string")
			}
			raw, err := jsonRaw(args[0])
			if err != nil {
				return nil, err
			}
			return fromResult(gjson.GetBytes(raw, path), raw, path), nil
		default:
			return nil, fmt.Errorf("json: invalid number of arguments")
		}
	}
}

func fnRaw(v any) (string, error) {
	raw, err := jsonRaw(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func jsonRaw(v any) ([]byte, error) {
	switch x := v.(type) {
	case value:
		if x.path == "" {
			return x.raw, nil
		}
		result := gjson.GetBytes(x.raw, x.path)
		if result.Type != gjson.JSON && result.Type != gjson.String {
			return nil, fmt.Errorf("json: value must be json")
		}
		return []byte(result.Raw), nil
	case string:
		return []byte(x), nil
	default:
		return nil, fmt.Errorf("json: value must be json")
	}
}

func fnTime(v any) (any, error) {
	n, ok := number(v)
	switch {
	case !ok:
		return nil, fmt.Errorf("time: value must be numeric")
	case n > 1_000_000_000_000 || n < -1_000_000_000_000:
		return time.Unix(0, n), nil
	default:
		return time.Unix(n, 0), nil
	}
}

func fnDuration(v string) (any, error) {
	return time.ParseDuration(v)
}

func number(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int64:
		return x, true
	case float64:
		return int64(x), true
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
