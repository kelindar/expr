package collection

import (
	"math"
	"testing"

	exprlib "github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionFunctions(t *testing.T) {
	got, err := Chunk([]any{1, 2, 3}, 2)
	require.NoError(t, err)
	require.Equal(t, [][]any{{1, 2}, {3}}, got)
	got, err = Chunk(nil, 2)
	require.NoError(t, err)
	require.Empty(t, got)
	got, err = Chunk(nil, math.MaxInt)
	require.NoError(t, err)
	require.Empty(t, got)
	_, err = Chunk([]any{1}, 0)
	require.Error(t, err)
	got, err = Zip([]any{1, 2}, []any{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, [][]any{{1, "a"}, {2, "b"}}, got)
	_, err = Zip([]any{1}, nil)
	require.Error(t, err)
	gotMap, err := Merge(map[string]any{"a": 1}, map[string]any{"a": 2, "b": 3})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"a": 2, "b": 3}, gotMap)
	gotAny, err := Union([]any{1, 2, 1}, []any{2.0, 3})
	require.NoError(t, err)
	require.Equal(t, []any{1, 2, 3}, gotAny)
	gotAny, err = Intersection([]any{1, 2, 2, 3}, []any{2.0, 4})
	require.NoError(t, err)
	require.Equal(t, []any{2}, gotAny)
	gotAny, err = Difference([]any{1, 2, 2, 3}, []any{2.0})
	require.NoError(t, err)
	require.Equal(t, []any{1, 3}, gotAny)
	gotAny, err = Intersection([]any{1, 2}, []any{3})
	require.NoError(t, err)
	require.Empty(t, gotAny)
	gotAny, err = Difference([]any{1, 1}, []any{3})
	require.NoError(t, err)
	require.Equal(t, []any{1}, gotAny)
	_, err = Union([]any{func() {}}, nil)
	require.Error(t, err)
	gotAny, err = Lag([]any{1, 2, 3}, 1)
	require.NoError(t, err)
	require.Equal(t, []any{nil, 1, 2}, gotAny)
	_, err = Lag(nil, -1)
	require.Error(t, err)
	gotFloat, err := Cumsum([]any{1, 2.5, int64(3)})
	require.NoError(t, err)
	require.Equal(t, []float64{1, 3.5, 6.5}, gotFloat)
	_, err = Cumsum([]any{"x"})
	require.Error(t, err)
	_, err = Cumsum([]any{math.Inf(1)})
	require.Error(t, err)
	_, err = Cumsum([]any{math.MaxFloat64, math.MaxFloat64})
	require.Error(t, err)
	gotFloat, err = Diff([]any{1, 3, 6})
	require.NoError(t, err)
	require.Equal(t, []float64{2, 3}, gotFloat)
	gotFloat, err = Diff([]any{1})
	require.NoError(t, err)
	require.Empty(t, gotFloat)
	_, err = Diff([]any{"x", 2})
	require.Error(t, err)
	_, err = Diff([]any{1, math.Inf(1)})
	require.Error(t, err)
	_, err = Diff([]any{-math.MaxFloat64, math.MaxFloat64})
	require.Error(t, err)
}

func TestConversions(t *testing.T) {
	got, err := array([2]int{1, 2})
	require.NoError(t, err)
	require.Equal(t, []any{1, 2}, got)
	_, err = array(1)
	require.Error(t, err)
	gotMap, err := object(map[string]any{"a": 1})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"a": 1}, gotMap)
	_, err = object([]any{})
	require.Error(t, err)
	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1)} {
		_, err := number(value)
		require.NoError(t, err)
	}
	_, err = number("x")
	require.Error(t, err)
	_, err = number(math.NaN())
	require.Error(t, err)
	_, err = integer(1.5)
	require.Error(t, err)
	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1)} {
		_, err := integer(value)
		require.NoError(t, err)
	}
	maxInt := int(^uint(0) >> 1)
	_, err = integer(uint64(maxInt) + 1)
	require.Error(t, err)
	_, err = integer(math.NaN())
	require.Error(t, err)
	_, err = integer(float64(maxInt) + 1)
	require.Error(t, err)
	gotInt, err := integer(int64(1<<53 + 1))
	require.NoError(t, err)
	require.Equal(t, int(1<<53+1), gotInt)
	gotInt, err = integer(uint64(1<<53 + 1))
	require.NoError(t, err)
	require.Equal(t, int(1<<53+1), gotInt)
	_, err = keys([]any{func() {}})
	require.Error(t, err)
	gotKey, err := key(map[string]any{"b": 2, "a": 1})
	require.NoError(t, err)
	require.Equal(t, `{"a":1,"b":2}`, gotKey)
	_, err = key(nil)
	require.NoError(t, err)
	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1), []any{int(1)}, map[string]any{"n": int(1)}} {
		require.NotNil(t, comparable(value))
	}
	gotNested := comparable(map[string]any{"n": int(1), "a": []any{uint(2)}})
	require.NotNil(t, gotNested)
}

func TestOptions(t *testing.T) {
	for _, source := range []string{
		`chunk([1, 2], 1)`, `zip([1], [2])`, `merge({"a": 1}, {"b": 2})`, `union([1], [2])`,
		`intersection([1], [1])`, `difference([1], [2])`, `lag([1], 0)`, `cumsum([1, 2])`, `diff([1, 2])`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		require.NoError(t, err, source)
		_, err = exprlib.Run(program, nil)
		require.NoError(t, err, source)
	}
	for _, source := range []string{
		`chunk()`, `chunk(1, 1)`, `chunk([1], "x")`, `zip()`, `zip(1, [1])`, `zip([1], 1)`,
		`merge()`, `merge(1, {})`, `merge({}, 1)`, `union()`, `union(1, [])`, `union([], 1)`,
		`intersection()`, `intersection(1, [])`, `intersection([], 1)`, `difference()`, `difference(1, [])`, `difference([], 1)`,
		`lag()`, `lag(1, 1)`, `lag([1], "x")`, `cumsum()`, `cumsum(1)`, `diff()`, `diff(1)`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		if err == nil {
			_, err = exprlib.Run(program, nil)
		}
		require.Error(t, err, source)
	}
}

func TestCollectionInvariants(t *testing.T) {
	input := []any{"a", "b", "c", "d", "e"}
	for _, size := range []int{1, 2, len(input), math.MaxInt} {
		t.Run("chunk", func(t *testing.T) {
			chunks, err := Chunk(input, size)
			require.NoError(t, err)
			var flattened []any
			for _, chunk := range chunks {
				assert.Greater(t, len(chunk), 0)
				assert.LessOrEqual(t, len(chunk), size)
				flattened = append(flattened, chunk...)
			}
			assert.Equal(t, input, flattened)
			if len(chunks) > 0 {
				chunks[0][0] = "changed"
				assert.Equal(t, "a", input[0])
			}
		})
	}

	left := []any{1, 2, 1, map[string]any{"a": []any{true}}}
	right := []any{2.0, 3, map[string]any{"a": []any{true}}}
	union, err := Union(left, right)
	require.NoError(t, err)
	assert.Equal(t, []any{1, 2, map[string]any{"a": []any{true}}, 3}, union)
	unionAgain, err := Union(union, nil)
	require.NoError(t, err)
	assert.Equal(t, union, unionAgain)

	intersection, err := Intersection(left, right)
	require.NoError(t, err)
	assert.Equal(t, []any{2, map[string]any{"a": []any{true}}}, intersection)
	difference, err := Difference(left, right)
	require.NoError(t, err)
	assert.Equal(t, []any{1}, difference)

	leftObject := map[string]any{"nested": map[string]any{"old": true}}
	rightObject := map[string]any{"value": 2}
	merged, err := Merge(leftObject, rightObject)
	require.NoError(t, err)
	merged["value"] = 3
	assert.NotContains(t, leftObject, "value")
	assert.Equal(t, 2, rightObject["value"])

	values := []any{10, 20, 35}
	for _, periods := range []int{0, 1, len(values), len(values) + 1} {
		lagged, err := Lag(values, periods)
		require.NoError(t, err)
		assert.Len(t, lagged, len(values))
		if periods == 0 {
			assert.Equal(t, values, lagged)
			lagged[0] = 99
			assert.Equal(t, 10, values[0])
		}
	}

	cumulative, err := Cumsum([]any{1, 2.5, 3})
	require.NoError(t, err)
	assert.Equal(t, []float64{1, 3.5, 6.5}, cumulative)
	deltas, err := Diff([]any{1, 3.5, 6.5})
	require.NoError(t, err)
	assert.Equal(t, []float64{2.5, 3}, deltas)
}
