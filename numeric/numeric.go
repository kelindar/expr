// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

// Package numeric provides deterministic numeric, statistical, and vector helpers.
package numeric

import (
	"fmt"
	"math"
	"sort"

	"gonum.org/v1/gonum/stat"
)

// Sqrt returns the square root of a finite, non-negative value.
func Sqrt(value float64) (float64, error) {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("sqrt: value must be finite and non-negative")
	}
	return math.Sqrt(value), nil
}

// Exp returns e raised to value when the finite result fits in a float64.
func Exp(value float64) (float64, error) {
	if !finite(value) {
		return 0, fmt.Errorf("exp: value must be finite")
	}
	out := math.Exp(value)
	if !finite(out) {
		return 0, fmt.Errorf("exp: result is not finite")
	}
	return out, nil
}

// Clamp limits value to the inclusive interval [low, high].
func Clamp(value, low, high float64) (float64, error) {
	if !finite(value) || !finite(low) || !finite(high) || low > high {
		return 0, fmt.Errorf("clamp: finite value and ordered bounds are required")
	}
	return min(high, max(low, value)), nil
}

// RoundTo rounds value to places decimal places, bounded to the supported range.
func RoundTo(value float64, places int) (float64, error) {
	if !finite(value) || places < -15 || places > 15 {
		return 0, fmt.Errorf("roundTo: finite value and places from -15 through 15 are required")
	}
	factor := math.Pow10(places)
	out := math.Round(value*factor) / factor
	if !finite(out) {
		return 0, fmt.Errorf("roundTo: result is not finite")
	}
	return out, nil
}

// Log returns the logarithm of value using the natural base or an optional base.
func Log(value float64, base ...float64) (float64, error) {
	if !finite(value) || value <= 0 {
		return 0, fmt.Errorf("log: value must be finite and positive")
	}
	b := math.E
	if len(base) > 1 {
		return 0, fmt.Errorf("log: expected zero or one base")
	}
	if len(base) == 1 {
		b = base[0]
		if !finite(b) || b <= 0 || b == 1 {
			return 0, fmt.Errorf("log: base must be finite, positive, and not one")
		}
	}
	return math.Log(value) / math.Log(b), nil
}

// Variance returns population or sample variance for finite values.
func Variance(values []float64, method string) (float64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("variance: values must not be empty")
	}
	if err := finiteValues(values); err != nil {
		return 0, err
	}
	switch method {
	case "", "population":
		return stat.PopVariance(values, nil), nil
	case "sample":
		if len(values) < 2 {
			return 0, fmt.Errorf("variance: sample requires at least two observations")
		}
		return stat.Variance(values, nil), nil
	default:
		return 0, fmt.Errorf("variance: method must be sample or population")
	}
}

// StdDev returns the population or sample standard deviation.
func StdDev(values []float64, method string) (float64, error) {
	variance, err := Variance(values, method)
	if err != nil {
		return 0, err
	}
	return math.Sqrt(variance), nil
}

// Quantile returns the linearly interpolated quantile at p in [0, 1].
func Quantile(values []float64, p float64) (float64, error) {
	switch {
	case len(values) == 0:
		return 0, fmt.Errorf("quantile: values must not be empty")
	case !finite(p) || p < 0 || p > 1:
		return 0, fmt.Errorf("quantile: p must be between zero and one")
	}
	if err := finiteValues(values); err != nil {
		return 0, err
	}
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	return stat.Quantile(p, stat.LinInterp, copyValues, nil), nil
}

// Covariance returns population or sample covariance for paired values.
func Covariance(x, y []float64, method string) (float64, error) {
	if err := pair(x, y, "covariance"); err != nil {
		return 0, err
	}
	if method == "" || method == "population" {
		return populationCovariance(x, y), nil
	}
	if method != "sample" {
		return 0, fmt.Errorf("covariance: method must be sample or population")
	}
	if len(x) < 2 {
		return 0, fmt.Errorf("covariance: sample requires at least two observations")
	}
	return stat.Covariance(x, y, nil), nil
}

// Correlation returns Pearson or Spearman correlation for paired values.
func Correlation(x, y []float64, method string) (float64, error) {
	if err := pair(x, y, "correlation"); err != nil {
		return 0, err
	}
	if method == "" || method == "pearson" {
		value := stat.Correlation(x, y, nil)
		if !finite(value) {
			return 0, fmt.Errorf("correlation: zero variance")
		}
		return value, nil
	}
	if method != "spearman" {
		return 0, fmt.Errorf("correlation: method must be pearson or spearman")
	}
	value := stat.Correlation(ranks(x), ranks(y), nil)
	if !finite(value) {
		return 0, fmt.Errorf("correlation: zero variance")
	}
	return value, nil
}

// Dot returns the dot product of two equally sized finite vectors.
func Dot(x, y []float64) (float64, error) {
	if err := pair(x, y, "dot"); err != nil {
		return 0, err
	}
	var out float64
	for i := range x {
		out += x[i] * y[i]
		if !finite(out) {
			return 0, fmt.Errorf("dot: result is not finite")
		}
	}
	return out, nil
}

// Norm returns the Euclidean norm of a non-empty finite vector.
func Norm(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("norm: values must not be empty")
	}
	if err := finiteValues(values); err != nil {
		return 0, err
	}
	var norm float64
	for _, value := range values {
		norm = math.Hypot(norm, value)
		if !finite(norm) {
			return 0, fmt.Errorf("norm: result is not finite")
		}
	}
	return norm, nil
}

// Normalize scales a finite vector to unit Euclidean norm.
func Normalize(values []float64) ([]float64, error) {
	norm, err := Norm(values)
	if err != nil {
		return nil, err
	}
	if norm == 0 {
		return nil, fmt.Errorf("normalize: zero norm")
	}
	out := make([]float64, len(values))
	for i, value := range values {
		out[i] = value / norm
		if !finite(out[i]) {
			return nil, fmt.Errorf("normalize: result is not finite")
		}
	}
	return out, nil
}

// Distance measures numeric vectors or strings with the selected distance method.
func Distance(x, y any, method string) (float64, error) {
	switch method {
	case "hamming":
		return hamming(x, y)
	case "levenshtein":
		return levenshteinDistance(x, y)
	}
	a, err := numbers(x)
	if err != nil {
		return 0, err
	}
	b, err := numbers(y)
	if err != nil {
		return 0, err
	}
	return numericDistance(a, b, method)
}

func numericDistance(x, y []float64, method string) (float64, error) {
	if err := pair(x, y, "distance"); err != nil {
		return 0, err
	}
	switch method {
	case "", "euclidean":
		var sum float64
		for i := range x {
			d := x[i] - y[i]
			sum += d * d
		}
		return math.Sqrt(sum), nil
	case "manhattan":
		var sum float64
		for i := range x {
			sum += math.Abs(x[i] - y[i])
		}
		return sum, nil
	case "chebyshev":
		var out float64
		for i := range x {
			out = max(out, math.Abs(x[i]-y[i]))
		}
		return out, nil
	default:
		return 0, fmt.Errorf("distance: method must be euclidean, manhattan, chebyshev, hamming, or levenshtein")
	}
}

// Similarity measures vectors, sets, or strings with the selected method.
func Similarity(x, y any, method string) (float64, error) {
	switch method {
	case "cosine":
		return cosineSimilarity(x, y)
	case "jaccard":
		return jaccard(x, y)
	case "hamming":
		return hamming(x, y)
	case "levenshtein":
		return levenshteinSimilarity(x, y)
	default:
		return 0, fmt.Errorf("similarity: method must be cosine, jaccard, hamming, or levenshtein")
	}
}

func cosineSimilarity(x, y any) (float64, error) {
	a, err := numbers(x)
	if err != nil {
		return 0, err
	}
	b, err := numbers(y)
	if err != nil {
		return 0, err
	}
	if err := pair(a, b, "similarity"); err != nil {
		return 0, err
	}
	na, err := Norm(a)
	if err != nil {
		return 0, err
	}
	nb, err := Norm(b)
	if err != nil {
		return 0, err
	}
	if na == 0 || nb == 0 {
		return 0, fmt.Errorf("similarity: cosine is undefined for a zero norm")
	}
	dot, err := Dot(a, b)
	if err != nil {
		return 0, err
	}
	denominator := na * nb
	if !finite(denominator) {
		return 0, fmt.Errorf("similarity: cosine result is not finite")
	}
	out := dot / denominator
	if !finite(out) {
		return 0, fmt.Errorf("similarity: cosine result is not finite")
	}
	return out, nil
}

func levenshteinSimilarity(x, y any) (float64, error) {
	a, ok := x.(string)
	if !ok {
		return 0, fmt.Errorf("similarity: levenshtein requires strings")
	}
	b, ok := y.(string)
	if !ok {
		return 0, fmt.Errorf("similarity: levenshtein requires strings")
	}
	left, right, err := levenshteinRunes(a, b)
	if err != nil {
		return 0, err
	}
	longest := max(len(left), len(right))
	if longest == 0 {
		return 1, nil
	}
	return 1 - float64(levenshtein(left, right))/float64(longest), nil
}
