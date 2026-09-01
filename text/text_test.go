// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package text

import (
	"fmt"
	"strings"
	"testing"

	exprlib "github.com/expr-lang/expr"
	"github.com/stretchr/testify/require"
)

func TestRegexAndUnicode(t *testing.T) {
	first, err := compile(`id-[0-9]+`)
	require.NoError(t, err)
	second, err := compile(`id-[0-9]+`)
	require.NoError(t, err)
	require.Same(t, first, second)
	for i := 0; i <= regexCacheSize; i++ {
		_, err := compile(fmt.Sprintf(`a{%d}`, i))
		require.NoError(t, err)
	}

	got, err := RegexFind("id-42", `(id-[0-9]+)`)
	require.NoError(t, err)
	require.Equal(t, []string{"id-42", "id-42"}, got)
	got, err = RegexFind("none", `id-[0-9]+`)
	require.NoError(t, err)
	require.Nil(t, got)
	gotAll, err := RegexFindAll("id-1 id-2", `id-[0-9]+`)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"id-1"}, {"id-2"}}, gotAll)
	gotText, err := RegexReplace("id-42", `id-([0-9]+)`, "item-$1")
	require.NoError(t, err)
	require.Equal(t, "item-42", gotText)
	for _, fn := range []func(string, string) error{
		func(value, pattern string) error { _, err := RegexFind(value, pattern); return err },
		func(value, pattern string) error { _, err := RegexFindAll(value, pattern); return err },
		func(value, pattern string) error { _, err := RegexReplace(value, pattern, "x"); return err },
	} {
		require.Error(t, fn("x", "["))
		require.Error(t, fn("x", strings.Repeat("a", maxPatternBytes+1)))
	}
	for _, form := range []string{"", "nfc", "nfd", "nfkc", "nfkd", "NFC"} {
		_, err := NormalizeUnicode("e\u0301", form)
		require.NoError(t, err)
	}
	_, err = NormalizeUnicode("x", "bad")
	require.Error(t, err)
	gotText, err = RemoveDiacritics("Crème brûlée")
	require.NoError(t, err)
	require.Equal(t, "Creme brulee", gotText)
}

func TestOptions(t *testing.T) {
	for _, source := range []string{
		`regexFind("id-42", "id-[0-9]+")`, `regexFindAll("id-42", "id-[0-9]+")`, `regexReplace("id-42", "id", "x")`,
		`normalizeUnicode("é")`, `normalizeUnicode("é", "nfd")`, `removeDiacritics("é")`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		require.NoError(t, err, source)
		_, err = exprlib.Run(program, nil)
		require.NoError(t, err, source)
	}
	for _, source := range []string{
		`regexFind()`, `regexFind(1, "x")`, `regexFind("x", 1)`, `regexFind("x", "[")`,
		`regexFindAll()`, `regexFindAll(1, "x")`, `regexFindAll("x", 1)`,
		`regexReplace()`, `regexReplace(1, "x", "y")`, `regexReplace("x", 1, "y")`, `regexReplace("x", "y", 1)`,
		`normalizeUnicode()`, `normalizeUnicode(1)`, `normalizeUnicode("x", 1)`, `normalizeUnicode("x", "bad")`,
		`removeDiacritics()`, `removeDiacritics(1)`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		if err == nil {
			_, err = exprlib.Run(program, nil)
		}
		require.Error(t, err, source)
	}
}

func FuzzRegexNoPanic(f *testing.F) {
	f.Add("id-42", `id-[0-9]+`)
	f.Add("", "[")
	f.Fuzz(func(t *testing.T, value, pattern string) {
		if len(value) > 1024 {
			value = value[:1024]
		}
		if len(pattern) > maxPatternBytes+32 {
			pattern = pattern[:maxPatternBytes+32]
		}
		require.NotPanics(t, func() {
			_, _ = RegexFind(value, pattern)
			_, _ = RegexFindAll(value, pattern)
			_, _ = RegexReplace(value, pattern, "replacement")
		})
	})
}
