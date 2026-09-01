// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package numeric

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"sort"
	"unicode/utf8"

	exprencoding "github.com/kelindar/expr/encoding"
)

func populationCovariance(x, y []float64) float64 {
	var sumX, sumY float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
	}
	meanX, meanY := sumX/float64(len(x)), sumY/float64(len(y))
	var out float64
	for i := range x {
		out += (x[i] - meanX) * (y[i] - meanY)
	}
	return out / float64(len(x))
}

func ranks(values []float64) []float64 {
	type pair struct {
		value float64
		index int
	}
	ordered := make([]pair, len(values))
	for i, value := range values {
		ordered[i] = pair{value, i}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].value < ordered[j].value })
	out := make([]float64, len(values))
	for i := 0; i < len(ordered); {
		j := i + 1
		for j < len(ordered) && ordered[j].value == ordered[i].value {
			j++
		}
		rank := (float64(i+1) + float64(j)) / 2
		for k := i; k < j; k++ {
			out[ordered[k].index] = rank
		}
		i = j
	}
	return out
}

func jaccard(x, y any) (float64, error) {
	a, err := values(x)
	if err != nil {
		return 0, err
	}
	b, err := values(y)
	if err != nil {
		return 0, err
	}
	left := make(map[string]struct{}, len(a))
	right := make(map[string]struct{}, len(b))
	for _, value := range a {
		key, err := jsonKey(value)
		if err != nil {
			return 0, err
		}
		left[string(key)] = struct{}{}
	}
	for _, value := range b {
		key, err := jsonKey(value)
		if err != nil {
			return 0, err
		}
		right[string(key)] = struct{}{}
	}
	if len(left) == 0 && len(right) == 0 {
		return 1, nil
	}
	intersection := 0
	for key := range left {
		if _, ok := right[key]; ok {
			intersection++
		}
	}
	return float64(intersection) / float64(len(left)+len(right)-intersection), nil
}

func hamming(x, y any) (float64, error) {
	if a, ok := x.(string); ok {
		b, ok := y.(string)
		if !ok {
			return 0, fmt.Errorf("similarity: hamming requires matching types")
		}
		ra, rb := []rune(a), []rune(b)
		if len(ra) != len(rb) {
			return 0, fmt.Errorf("similarity: hamming strings must have equal lengths")
		}
		count := 0
		for i := range ra {
			if ra[i] != rb[i] {
				count++
			}
		}
		return float64(count), nil
	}
	a, err := values(x)
	if err != nil {
		return 0, err
	}
	b, err := values(y)
	if err != nil {
		return 0, err
	}
	if len(a) != len(b) {
		return 0, fmt.Errorf("similarity: hamming arrays must have equal lengths")
	}
	count := 0
	for i := range a {
		equal, err := equalJSON(a[i], b[i])
		if err != nil {
			return 0, fmt.Errorf("similarity: hamming index %d: %w", i, err)
		}
		if !equal {
			count++
		}
	}
	return float64(count), nil
}

func equalJSON(left, right any) (bool, error) {
	a, err := jsonKey(left)
	if err != nil {
		return false, err
	}
	b, err := jsonKey(right)
	if err != nil {
		return false, err
	}
	return bytes.Equal(a, b), nil
}

func jsonKey(value any) ([]byte, error) {
	raw, err := json.Marshal(value, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return exprencoding.CanonicalJSON(raw)
}

func levenshteinDistance(x, y any) (float64, error) {
	a, ok := x.(string)
	if !ok {
		return 0, fmt.Errorf("distance: levenshtein requires strings")
	}
	b, ok := y.(string)
	if !ok {
		return 0, fmt.Errorf("distance: levenshtein requires strings")
	}
	left, right, err := levenshteinRunes(a, b)
	if err != nil {
		return 0, err
	}
	return float64(levenshtein(left, right)), nil
}

const (
	maxLevenshteinRunes = 16_384
	maxLevenshteinWork  = 4_000_000
)

func checkLevenshtein(a, b []rune) error {
	if len(a) > maxLevenshteinRunes || len(b) > maxLevenshteinRunes {
		return fmt.Errorf("distance: levenshtein input exceeds %d runes", maxLevenshteinRunes)
	}
	if len(a) != 0 && len(b) > maxLevenshteinWork/len(a) {
		return fmt.Errorf("distance: levenshtein work exceeds %d cell updates", maxLevenshteinWork)
	}
	return nil
}

func levenshteinRunes(a, b string) ([]rune, []rune, error) {
	if utf8.RuneCountInString(a) > maxLevenshteinRunes || utf8.RuneCountInString(b) > maxLevenshteinRunes {
		return nil, nil, fmt.Errorf("distance: levenshtein input exceeds %d runes", maxLevenshteinRunes)
	}
	left, right := []rune(a), []rune(b)
	if err := checkLevenshtein(left, right); err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

func values(value any) ([]any, error) {
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

func levenshtein(a, b []rune) int {
	if len(a) < len(b) {
		a, b = b, a
	}
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for i := range previous {
		previous[i] = i
	}
	for i, ra := range a {
		current[0] = i + 1
		for j, rb := range b {
			cost := 0
			if ra != rb {
				cost = 1
			}
			current[j+1] = min(current[j]+1, min(previous[j+1]+1, previous[j]+cost))
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}
