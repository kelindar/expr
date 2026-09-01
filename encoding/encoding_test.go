// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package encoding

import (
	"testing"

	exprlib "github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalHashes(t *testing.T) {
	got, err := CanonicalJSON([]byte(`{"b":2,"a":1}`))
	require.NoError(t, err)
	require.Equal(t, `{"a":1,"b":2}`, string(got))
	for _, raw := range [][]byte{nil, []byte(`{`)} {
		_, err := CanonicalJSON(raw)
		require.Error(t, err)
	}
	require.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", mustHash("hello", "sha256"))
	for _, algorithm := range []string{"md5", "sha1", "sha256", "sha384", "sha512", "xxh3", "SHA256"} {
		_, err := Hash("hello", algorithm)
		require.NoError(t, err)
	}
	_, err = Hash(map[string]any{"b": 2, "a": 1}, "sha256")
	require.NoError(t, err)
	_, err = Hash("hello", "bad")
	require.Error(t, err)
	_, err = Hash(func() {}, "sha256")
	require.Error(t, err)
	gotChecksum, err := Checksum("hello", "crc32")
	require.NoError(t, err)
	require.Equal(t, "3610a686", gotChecksum)
	_, err = Checksum("hello", "bad")
	require.Error(t, err)
}

func TestHexAndURL(t *testing.T) {
	require.Equal(t, "6869", HexEncode("hi"))
	got, err := HexDecode("6869")
	require.NoError(t, err)
	require.Equal(t, "hi", got)
	for _, value := range []string{"x", "ff"} {
		_, err := HexDecode(value)
		require.Error(t, err)
	}
	for _, mode := range []string{"query", "path_segment"} {
		encoded, err := URLEncode("a b/c", mode)
		require.NoError(t, err)
		_, err = URLDecode(encoded, mode)
		require.NoError(t, err)
	}
	_, err = URLEncode("x", "bad")
	require.Error(t, err)
	_, err = URLDecode("x", "bad")
	require.Error(t, err)
	_, err = URLDecode("%zz", "query")
	require.Error(t, err)
}

func mustHash(value any, algorithm string) string {
	got, err := Hash(value, algorithm)
	if err != nil {
		panic(err)
	}
	return got
}

func TestOptions(t *testing.T) {
	for _, source := range []string{
		`hash("hello", "sha256")`, `checksum("hello", "crc32")`, `hexEncode("hi")`, `hexDecode("6869")`,
		`urlEncode("a b", "query")`, `urlDecode("a+b", "query")`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		require.NoError(t, err, source)
		_, err = exprlib.Run(program, nil)
		require.NoError(t, err, source)
	}
	for _, source := range []string{
		`hash()`, `hash("x")`, `hash("x", 1)`, `checksum()`, `checksum("x")`, `checksum("x", 1)`,
		`hexEncode()`, `hexEncode(1)`, `hexDecode()`, `hexDecode(1)`, `hexDecode("ff")`,
		`urlEncode()`, `urlEncode(1, "query")`, `urlEncode("x", 1)`, `urlEncode("x", "bad")`,
		`urlDecode()`, `urlDecode(1, "query")`, `urlDecode("x", 1)`, `urlDecode("%zz", "query")`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		if err == nil {
			_, err = exprlib.Run(program, nil)
		}
		require.Error(t, err, source)
	}
}

func TestEncodingInvariants(t *testing.T) {
	first, err := CanonicalJSON([]byte(`{"z":3,"a":{"y":2,"x":1}}`))
	require.NoError(t, err)
	second, err := CanonicalJSON([]byte(`{"a":{"x":1,"y":2},"z":3}`))
	require.NoError(t, err)
	assert.Equal(t, first, second)
	again, err := CanonicalJSON(first)
	require.NoError(t, err)
	assert.Equal(t, first, again)

	left := map[string]any{"z": 3, "a": map[string]any{"x": 1}}
	right := map[string]any{"a": map[string]any{"x": 1}, "z": 3}
	for _, algorithm := range []string{"md5", "sha1", "sha256", "sha384", "sha512", "xxh3"} {
		leftHash, err := Hash(left, algorithm)
		require.NoError(t, err)
		rightHash, err := Hash(right, algorithm)
		require.NoError(t, err)
		assert.Equal(t, leftHash, rightHash, algorithm)
	}
	leftChecksum, err := Checksum(left, "crc32")
	require.NoError(t, err)
	rightChecksum, err := Checksum(right, "crc32")
	require.NoError(t, err)
	assert.Equal(t, leftChecksum, rightChecksum)

	for _, value := range []string{"hello world", "a/b?c=1", "café/東京"} {
		for _, mode := range []string{"query", "path_segment"} {
			encoded, err := URLEncode(value, mode)
			require.NoError(t, err)
			decoded, err := URLDecode(encoded, mode)
			require.NoError(t, err)
			assert.Equal(t, value, decoded)
		}
	}

	for _, value := range []string{"", "hello", "é"} {
		encoded := HexEncode(value)
		decoded, err := HexDecode(encoded)
		require.NoError(t, err)
		assert.Equal(t, value, decoded)
	}
}

func FuzzCanonicalJSONNoPanic(f *testing.F) {
	for _, seed := range []string{`{"b":2,"a":1}`, `[]`, `"text"`, `{`, `1e999`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 4096 {
			raw = raw[:4096]
		}
		require.NotPanics(t, func() {
			_, _ = CanonicalJSON([]byte(raw))
		})
	})
}
