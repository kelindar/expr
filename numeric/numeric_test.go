// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package numeric

import (
	"encoding/json/v2"
	"math"
	"strings"
	"testing"

	exprlib "github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScalarFunctions(t *testing.T) {
	got, err := Sqrt(9)
	require.NoError(t, err)
	require.Equal(t, 3.0, got)
	for _, value := range []float64{-1, math.NaN(), math.Inf(1)} {
		_, err := Sqrt(value)
		require.Error(t, err)
	}
	got, err = Exp(1)
	require.NoError(t, err)
	require.InDelta(t, math.E, got, 1e-12)
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := Exp(value)
		require.Error(t, err)
	}
	_, err = Exp(1000)
	require.Error(t, err)
	got, err = Clamp(2, 0, 1)
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	got, err = Clamp(-1, 0, 1)
	require.NoError(t, err)
	require.Equal(t, 0.0, got)
	for _, tc := range [][3]float64{{math.NaN(), 0, 1}, {1, 2, 1}, {1, math.Inf(1), 2}} {
		_, err := Clamp(tc[0], tc[1], tc[2])
		require.Error(t, err)
	}
	got, err = RoundTo(1.234, 2)
	require.NoError(t, err)
	require.Equal(t, 1.23, got)
	for _, places := range []int{-16, 16} {
		_, err := RoundTo(1, places)
		require.Error(t, err)
	}
	_, err = RoundTo(math.NaN(), 1)
	require.Error(t, err)
	_, err = RoundTo(math.MaxFloat64, 15)
	require.Error(t, err)
	got, err = Log(100, 10)
	require.NoError(t, err)
	require.Equal(t, 2.0, got)
	_, err = Log(10)
	require.NoError(t, err)
	for _, tc := range []struct {
		value float64
		base  []float64
	}{
		{0, nil}, {-1, nil}, {1, []float64{1, 2}}, {10, []float64{0}}, {10, []float64{1}}, {10, []float64{math.NaN()}},
	} {
		_, err := Log(tc.value, tc.base...)
		require.Error(t, err)
	}
}

func TestStatisticsAndVectors(t *testing.T) {
	values := []float64{1, 2, 3}
	got, err := Variance(values, "")
	require.NoError(t, err)
	require.Equal(t, 2.0/3, got)
	got, err = Variance(values, "sample")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	for _, tc := range []struct {
		values []float64
		method string
	}{
		{nil, ""}, {values[:1], "sample"}, {values, "bad"}, {[]float64{math.NaN()}, ""},
	} {
		_, err := Variance(tc.values, tc.method)
		require.Error(t, err)
	}
	got, err = StdDev(values, "")
	require.NoError(t, err)
	require.InDelta(t, math.Sqrt(2.0/3), got, 1e-12)
	_, err = StdDev(nil, "")
	require.Error(t, err)
	got, err = Quantile([]float64{4, 1, 3, 2}, .5)
	require.NoError(t, err)
	require.Equal(t, 2.0, got)
	for _, p := range []float64{-0.1, 1.1, math.NaN()} {
		_, err := Quantile(values, p)
		require.Error(t, err)
	}
	_, err = Quantile(nil, .5)
	require.Error(t, err)
	_, err = Quantile([]float64{math.Inf(1)}, .5)
	require.Error(t, err)

	got, err = Covariance(values, []float64{2, 4, 6}, "")
	require.NoError(t, err)
	require.Equal(t, 4.0/3, got)
	got, err = Covariance(values, []float64{2, 4, 6}, "sample")
	require.NoError(t, err)
	require.Equal(t, 2.0, got)
	for _, tc := range []struct {
		x, y   []float64
		method string
	}{
		{nil, values, ""}, {values[:2], values, ""}, {values, values, "bad"}, {values[:1], values[:1], "sample"}, {[]float64{math.NaN()}, []float64{1}, ""},
	} {
		_, err := Covariance(tc.x, tc.y, tc.method)
		require.Error(t, err)
	}
	got, err = Correlation(values, []float64{2, 4, 6}, "")
	require.NoError(t, err)
	got, err = Correlation(values, []float64{2, 4, 6}, "pearson")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	got, err = Correlation([]float64{3, 1, 2}, []float64{30, 10, 20}, "spearman")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	for _, method := range []string{"bad"} {
		_, err := Correlation(values, values, method)
		require.Error(t, err)
	}
	_, err = Correlation([]float64{1, 1}, []float64{2, 3}, "")
	require.Error(t, err)
	_, err = Correlation([]float64{1, 1}, []float64{2, 2}, "spearman")
	require.Error(t, err)
	_, err = Correlation([]float64{1}, []float64{1, 2}, "")
	require.Error(t, err)

	got, err = Dot([]float64{1, 2}, []float64{3, 4})
	require.NoError(t, err)
	require.Equal(t, 11.0, got)
	_, err = Dot([]float64{math.MaxFloat64}, []float64{2})
	require.Error(t, err)
	_, err = Dot([]float64{1}, []float64{1, 2})
	require.Error(t, err)
	got, err = Norm([]float64{3, 4})
	require.NoError(t, err)
	require.Equal(t, 5.0, got)
	_, err = Norm(nil)
	require.Error(t, err)
	_, err = levenshteinDistance("abc", "ac")
	require.NoError(t, err)
	_, err = levenshteinDistance("abc", 1)
	require.Error(t, err)
	_, err = levenshteinDistance("", strings.Repeat("a", maxLevenshteinRunes+1))
	require.Error(t, err)
	_, err = hamming([]any{1}, []any{1, 2})
	require.Error(t, err)
	require.Equal(t, []float64{1.5, 1.5, 3}, ranks([]float64{1, 1, 3}))
	_, err = Norm([]float64{math.Inf(1)})
	require.Error(t, err)
	_, err = Normalize([]float64{math.MaxFloat64, math.MaxFloat64})
	require.Error(t, err)
	gotVec, err := Normalize([]float64{3, 4})
	require.NoError(t, err)
	require.Equal(t, []float64{.6, .8}, gotVec)
	_, err = Normalize([]float64{0})
	require.Error(t, err)
	for _, tc := range []struct {
		x, y   []float64
		method string
	}{
		{[]float64{1, 2}, []float64{3}, ""}, {nil, nil, ""}, {[]float64{1}, []float64{2}, "bad"}, {[]float64{1}, []float64{2}, "euclidean"}, {[]float64{1}, []float64{2}, "manhattan"}, {[]float64{1}, []float64{2}, "chebyshev"}, {[]float64{1}, []float64{2}, "hamming"}, {[]float64{1}, []float64{2}, "levenshtein"},
	} {
		_, err := Distance(tc.x, tc.y, tc.method)
		if tc.method == "bad" || tc.x == nil || len(tc.x) != len(tc.y) || tc.method == "levenshtein" {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
	got, err = Distance("abc", "adc", "hamming")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	got, err = Distance("abc", "ac", "levenshtein")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
}

func TestSimilarityAndHelpers(t *testing.T) {
	got, err := Similarity([]float64{1, 0}, []float64{1, 0}, "cosine")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	got, err = Similarity([]any{1, 2}, []any{2, 3}, "jaccard")
	require.NoError(t, err)
	require.Equal(t, 1.0/3, got)
	got, err = Similarity([]any{}, []any{}, "jaccard")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	got, err = Similarity("abc", "adc", "hamming")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	got, err = Similarity("abc", "abd", "levenshtein")
	require.NoError(t, err)
	require.InDelta(t, 2.0/3, got, 1e-12)
	got, err = Similarity("", "", "levenshtein")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	got, err = Similarity([]any{map[string]any{"a": []any{1}}}, []any{map[string]any{"a": []any{1}}}, "hamming")
	require.NoError(t, err)
	require.Equal(t, 0.0, got)
	got, err = Similarity([]any{map[string]any{"a": []any{1}}}, []any{map[string]any{"a": []any{2}}}, "hamming")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	_, err = Similarity(strings.Repeat("a", 2049), strings.Repeat("b", 2049), "levenshtein")
	require.Error(t, err)
	for _, tc := range []struct {
		x, y any
		m    string
	}{
		{[]float64{0}, []float64{1}, "cosine"}, {[]float64{1}, []float64{1, 2}, "cosine"}, {"a", "bb", "hamming"}, {"a", 1, "hamming"}, {1, "a", "levenshtein"}, {1, 2, "bad"},
	} {
		_, err := Similarity(tc.x, tc.y, tc.m)
		require.Error(t, err)
	}
	_, err = Similarity([]any{func() {}}, []any{}, "jaccard")
	require.Error(t, err)
	_, err = Similarity([]any{func() {}}, []any{func() {}}, "hamming")
	require.Error(t, err)
	_, err = Similarity([]any{1}, []any{2}, "jaccard")
	require.NoError(t, err)
	got, err = Similarity([]any{map[string]any{"value": 1}}, []any{map[string]any{"value": 1.0}}, "jaccard")
	require.NoError(t, err)
	require.Equal(t, 1.0, got)
	_, err = Similarity([]float64{1}, []float64{2}, "hamming")
	require.NoError(t, err)
	_, err = Similarity([]float64{math.MaxFloat64}, []float64{2}, "cosine")
	require.Error(t, err)
	_, err = Similarity([]float64{math.MaxFloat64, 0}, []float64{0, math.MaxFloat64}, "cosine")
	require.Error(t, err)
	_, err = Similarity("a", 1, "levenshtein")
	require.Error(t, err)

	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1)} {
		_, err := number(value)
		require.NoError(t, err)
	}
	_, err = number("x")
	require.Error(t, err)
	_, err = finiteNumber(math.NaN())
	require.Error(t, err)
	gotInt, err := integer(float64(2))
	require.NoError(t, err)
	require.Equal(t, 2, gotInt)
	gotInt, err = integer(int64(1<<53 + 1))
	require.NoError(t, err)
	require.Equal(t, int(1<<53+1), gotInt)
	gotInt, err = integer(uint64(1<<53 + 1))
	require.NoError(t, err)
	require.Equal(t, int(1<<53+1), gotInt)
	for _, value := range []any{int8(1), int16(1), int32(1), uint(1), uint8(1), uint16(1), uint32(1), float32(1)} {
		_, err = integer(value)
		require.NoError(t, err)
	}
	_, err = integer(1.5)
	require.Error(t, err)
	_, err = oneNumber(nil, "x")
	require.Error(t, err)
	_, err = text(1)
	require.Error(t, err)
	_, err = numbers(1)
	require.Error(t, err)
	_, err = numbers([]any{1, "x"})
	require.Error(t, err)
	require.Equal(t, []float64{1, 2}, mustNumbers([]int{1, 2}))
	require.Error(t, pair([]float64{1}, []float64{math.NaN()}, "pair"))
	require.Equal(t, []float64{1, 2}, ranks([]float64{3, 4}))
	require.Equal(t, 1, levenshtein([]rune("a"), []rune("")))
	_, err = levenshteinDistance(1, "a")
	require.Error(t, err)
	_, err = levenshteinDistance(strings.Repeat("a", maxLevenshteinRunes+1), "a")
	require.Error(t, err)
	require.Equal(t, 1, levenshtein([]rune(""), []rune("a")))
	err = checkLevenshtein(nil, nil)
	require.NoError(t, err)
	err = checkLevenshtein([]rune("a"), []rune(strings.Repeat("a", maxLevenshteinWork)))
	require.Error(t, err)
	_, err = logFunc(10)
	require.NoError(t, err)
	_, err = distanceFunc([]float64{1}, []float64{1}, 1)
	require.Error(t, err)
	_, err = similarityFunc([]float64{1}, []float64{1}, 1)
	require.Error(t, err)
	_, err = jaccard(1, []any{})
	require.Error(t, err)
	_, err = jaccard([]any{}, 1)
	require.Error(t, err)
	_, err = jaccard([]any{}, []any{func() {}})
	require.Error(t, err)
	_, err = hamming(1, []any{})
	require.Error(t, err)
	_, err = hamming([]any{}, 1)
	require.Error(t, err)
	_, err = hamming([]any{func() {}}, []any{1})
	require.Error(t, err)
	_, err = values(1)
	require.Error(t, err)
}

func mustNumbers(value any) []float64 {
	out, err := numbers(value)
	if err != nil {
		panic(err)
	}
	return out
}

func TestOptions(t *testing.T) {
	for _, source := range []string{
		"sqrt(9)", "exp(1)", "clamp(2, 0, 1)", "roundTo(1.234, 2)", "log(100, 10)",
		"variance([1, 2, 3])", "stddev([1, 2, 3], \"sample\")", "quantile([1, 2, 3], 0.5)",
		"covariance([1, 2], [2, 4])", "correlation([1, 2], [2, 4])", "dot([1, 2], [3, 4])",
		"norm([3, 4])", "normalize([3, 4])", "distance([1, 2], [3, 4], \"manhattan\")", "distance(\"abc\", \"adc\", \"hamming\")", "distance(\"abc\", \"ac\", \"levenshtein\")", "similarity([1, 0], [1, 0], \"cosine\")",
	} {
		program, err := exprlib.Compile(source, Options()...)
		require.NoError(t, err, source)
		_, err = exprlib.Run(program, nil)
		require.NoError(t, err, source)
	}
	for _, source := range []string{"sqrt()", "clamp(1)", "roundTo(1)", "log()", "variance()", "quantile([1], 2)", "covariance([1])", "correlation([1])", "dot([1])", "norm()", "normalize()", "distance([1])", "similarity(1, 2, \"bad\")"} {
		program, err := exprlib.Compile(source, Options()...)
		if err == nil {
			_, err = exprlib.Run(program, nil)
		}
		require.Error(t, err, source)
	}
	for _, source := range []string{
		`exp("x")`, `clamp("x", 0, 1)`, `clamp(1, "x", 2)`, `clamp(1, 0, "x")`,
		`roundTo("x", 2)`, `roundTo(1, "x")`, `log("x")`, `log(1, "x")`,
		`variance([1], "sample")`, `variance([1], "bad")`, `stddev([1], "bad")`,
		`quantile("x", 0.5)`, `quantile([1], "x")`, `covariance("x", [1])`, `covariance([1], "x")`, `covariance([1], [1], "bad")`,
		`correlation("x", [1])`, `correlation([1], "x")`, `correlation([1], [1], "bad")`, `dot("x", [1])`, `dot([1], "x")`,
		`norm("x")`, `normalize("x")`, `distance("x", [1])`, `distance([1], "x")`, `distance([1], [1], "bad")`,
		`similarity("x", "y", "cosine")`, `similarity([1], "x", "cosine")`, `similarity([1], [1, 2], "cosine")`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		if err == nil {
			_, err = exprlib.Run(program, nil)
		}
		require.Error(t, err, source)
	}
}

func TestFunctionArgumentErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func() (any, error)
	}{
		{"log value", func() (any, error) { return logFunc([]any{"x"}) }},
		{"log base", func() (any, error) { return logFunc([]any{1, "x"}) }},
		{"stddev method", func() (any, error) { return stddevFunc([]any{[]float64{1, 2}, "bad"}) }},
		{"quantile p", func() (any, error) { return quantileFunc([]any{[]float64{1, 2}, "x"}) }},
		{"distance arity", func() (any, error) { return distanceFunc([]any{1}) }},
		{"distance method", func() (any, error) { return distanceFunc([]any{[]float64{1}, []float64{1}, "bad"}) }},
		{"similarity arity", func() (any, error) { return similarityFunc([]any{1, 2}) }},
		{"similarity method", func() (any, error) { return similarityFunc([]any{[]float64{1}, []float64{1}, 1}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.call()
			require.Error(t, err)
		})
	}
}

func TestNumericInvariants(t *testing.T) {
	vectors := [][]float64{{3, 4}, {-1, 2, 3}, {0.5}}
	for _, vector := range vectors {
		original := append([]float64(nil), vector...)
		normalized, err := Normalize(vector)
		require.NoError(t, err)
		length, err := Norm(normalized)
		require.NoError(t, err)
		assert.InDelta(t, 1, length, 1e-12)
		assert.Equal(t, original, vector)
		norm, err := Norm(vector)
		require.NoError(t, err)
		dot, err := Dot(vector, vector)
		require.NoError(t, err)
		assert.InDelta(t, norm*norm, dot, 1e-12)
	}

	left := []float64{1, 2, 3}
	right := []float64{3, 1, 2}
	for _, method := range []string{"", "euclidean", "manhattan", "chebyshev"} {
		forward, err := Distance(left, right, method)
		require.NoError(t, err)
		backward, err := Distance(right, left, method)
		require.NoError(t, err)
		assert.InDelta(t, forward, backward, 1e-12)
		assert.GreaterOrEqual(t, forward, 0.0)
	}
	cosine, err := Similarity(left, left, "cosine")
	require.NoError(t, err)
	assert.InDelta(t, 1, cosine, 1e-12)
	cosine, err = Similarity(left, right, "cosine")
	require.NoError(t, err)
	cosineBack, err := Similarity(right, left, "cosine")
	require.NoError(t, err)
	assert.InDelta(t, cosine, cosineBack, 1e-12)

	for _, pair := range [][2]string{{"kitten", "sitting"}, {"", "abc"}, {"naïve", "naive"}} {
		forward, err := Distance(pair[0], pair[1], "levenshtein")
		require.NoError(t, err)
		backward, err := Distance(pair[1], pair[0], "levenshtein")
		require.NoError(t, err)
		assert.Equal(t, forward, backward)
		assert.GreaterOrEqual(t, forward, 0.0)
	}

	hamming, err := Similarity([]any{map[string]any{"items": []any{1, 2}}}, []any{map[string]any{"items": []any{1, 2}}}, "hamming")
	require.NoError(t, err)
	assert.Equal(t, 0.0, hamming)
	hamming, err = Similarity([]any{[]any{1, map[string]any{"ok": true}}}, []any{[]any{1, map[string]any{"ok": false}}}, "hamming")
	require.NoError(t, err)
	assert.Equal(t, 1.0, hamming)

	jaccard, err := Similarity([]any{1, 1, 2}, []any{2.0, 3}, "jaccard")
	require.NoError(t, err)
	assert.InDelta(t, 1.0/3, jaccard, 1e-12)
	assert.Equal(t, jaccard, mustSimilarity([]any{2.0, 3}, []any{1, 1, 2}, "jaccard"))

	values := []float64{1, 2, 3, 4}
	population, err := Variance(values, "population")
	require.NoError(t, err)
	sample, err := Variance(values, "sample")
	require.NoError(t, err)
	assert.InDelta(t, population*float64(len(values))/float64(len(values)-1), sample, 1e-12)
	low, err := Quantile(values, 0)
	require.NoError(t, err)
	high, err := Quantile(values, 1)
	require.NoError(t, err)
	assert.Equal(t, 1.0, low)
	assert.Equal(t, 4.0, high)
}

func mustSimilarity(left, right any, method string) float64 {
	value, err := Similarity(left, right, method)
	if err != nil {
		panic(err)
	}
	return value
}

func FuzzSimilarityNoPanic(f *testing.F) {
	f.Add(`[1,{"a":[2]}]`, `[1,{"a":[3]}]`)
	f.Add(`[{"a":1}]`, `[{"a":1}]`)
	f.Fuzz(func(t *testing.T, leftRaw, rightRaw string) {
		if len(leftRaw) > 1024 {
			leftRaw = leftRaw[:1024]
		}
		if len(rightRaw) > 1024 {
			rightRaw = rightRaw[:1024]
		}
		var left, right any
		if err := json.Unmarshal([]byte(leftRaw), &left); err != nil {
			return
		}
		if err := json.Unmarshal([]byte(rightRaw), &right); err != nil {
			return
		}
		require.NotPanics(t, func() {
			_, _ = Similarity(left, right, "hamming")
			_, _ = Similarity(left, right, "jaccard")
		})
	})
}

func FuzzLevenshteinBounded(f *testing.F) {
	f.Add("kitten", "sitting")
	f.Add("", "a")
	f.Fuzz(func(t *testing.T, left, right string) {
		if len(left) > 256 {
			left = left[:256]
		}
		if len(right) > 256 {
			right = right[:256]
		}
		require.NotPanics(t, func() {
			_, _ = Distance(left, right, "levenshtein")
			_, _ = Similarity(left, right, "levenshtein")
		})
	})
}
