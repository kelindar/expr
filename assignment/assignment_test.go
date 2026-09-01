// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package assignment

import (
	"math"
	"testing"

	exprlib "github.com/expr-lang/expr"
	"github.com/stretchr/testify/require"
)

func TestBucket(t *testing.T) {
	left, err := Bucket(map[string]any{"b": 2, "a": 1}, 100)
	require.NoError(t, err)
	right, err := Bucket(map[string]any{"a": 1, "b": 2}, 100)
	require.NoError(t, err)
	require.Equal(t, left, right)
	namespaced, err := Bucket("user-42", 10, "checkout")
	require.NoError(t, err)
	require.GreaterOrEqual(t, namespaced, 0)
	require.Less(t, namespaced, 10)
	for _, tc := range []struct {
		value any
		count int
		ns    []string
	}{
		{"x", 0, nil}, {"x", -1, nil}, {"x", 1, []string{"a", "b"}}, {func() {}, 10, nil},
	} {
		_, err := Bucket(tc.value, tc.count, tc.ns...)
		require.Error(t, err)
	}
}

func TestInteger(t *testing.T) {
	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1)} {
		_, err := integer(value)
		require.NoError(t, err)
	}
	_, err := integer(1.5)
	require.Error(t, err)
	maxInt := int(^uint(0) >> 1)
	_, err = integer(uint64(maxInt) + 1)
	require.Error(t, err)
	_, err = integer(float64(math.MaxInt64))
	require.Error(t, err)
	for _, value := range []any{int8(1), int16(1), int32(1), uint(1), uint8(1), uint16(1), uint32(1)} {
		_, err = integer(value)
		require.NoError(t, err)
	}
}

func TestOptions(t *testing.T) {
	program, err := exprlib.Compile(`bucket("user-42", 10, "checkout")`, Options()...)
	require.NoError(t, err)
	_, err = exprlib.Run(program, nil)
	require.NoError(t, err)
	program, err = exprlib.Compile(`bucket("user-42", 0)`, Options()...)
	require.NoError(t, err)
	_, err = exprlib.Run(program, nil)
	require.Error(t, err)
	program, err = exprlib.Compile(`bucket("user-42")`, Options()...)
	if err == nil {
		_, err = exprlib.Run(program, nil)
	}
	require.Error(t, err)
	program, err = exprlib.Compile(`bucket("user-42", 10, 1)`, Options()...)
	if err == nil {
		_, err = exprlib.Run(program, nil)
	}
	require.Error(t, err)
}

func TestBucketFunctionValidation(t *testing.T) {
	_, err := bucket("only-one")
	require.EqualError(t, err, "bucket: expected two or three arguments")
	_, err = bucket("x", 10, 1)
	require.EqualError(t, err, "bucket: namespace must be a string")
	_, err = bucket("x", 1.5)
	require.EqualError(t, err, "bucket: count must be an integer")
}
