// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package validate

import (
	"encoding/json/jsontext"
	"math"
	"strings"
	"testing"

	exprlib "github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRulesAndPredicates(t *testing.T) {
	valid := map[string]any{
		"null": nil, "boolean": true, "number": 1.5, "integer": int64(2), "string": "hello",
		"array": []any{1}, "object": map[string]any{"a": 1}, "json": map[string]any{"a": 1},
		"base64": "aGk=", "data_uri": "data:text/plain,hello", "email": "user@example.com", "url": "https://example.com",
		"request_url": "https://example.com/a", "uri": "urn:example:x", "hex": "deadbeef", "uuid": "550e8400-e29b-41d4-a716-446655440000",
		"uuid_v1": "550e8400-e29b-11d4-a716-446655440000", "uuid_v3": "550e8400-e29b-31d4-a716-446655440000", "uuid_v4": "550e8400-e29b-41d4-a716-446655440000", "uuid_v5": "550e8400-e29b-51d4-a716-446655440000",
		"mongo_id": "507f1f77bcf86cd799439011", "imei": "490154203237518", "imsi": "310150123456789", "ip": "10.0.0.1", "ip_v4": "10.0.0.1", "ip_v6": "2001:db8::1", "cidr": "10.0.0.0/8",
		"dns": "example.com", "hostname": "example.com", "mac": "00:11:22:33:44:55", "port": int64(443), "rfc3339": "2026-08-30T12:00:00Z", "rfc3339_without_zone": "2026-08-30T12:00:00",
		"semver": "1.2.3", "iso_country_2": "AE", "iso_country_3": "ARE", "iso_currency": "AED", "latitude": 25.2, "longitude": 55.3,
		"hash": "0123456789abcdef0123456789abcdef", "crc32": "3610a686", "regex_syntax": "[a-z]+", "regex": "[a-z]+", "printable_ascii": "hello", "unicode_letter": "Привет", "unicode_digit": "१२३", "color": "#ff00aa", "phone": "+971 50 123 4567",
	}
	rulesList := Rules()
	require.NotEmpty(t, rulesList)
	assert.Len(t, rulesList, len(rules))
	assert.Len(t, ruleDocs, len(rules))
	for i := 1; i < len(rulesList); i++ {
		assert.LessOrEqual(t, rulesList[i-1].Name, rulesList[i].Name)
	}
	for _, rule := range rulesList {
		assert.NotEmpty(t, rule.Summary, rule.Name)
		assert.NotEmpty(t, rule.Example, rule.Name)
		value, ok := valid[rule.Name]
		require.True(t, ok, rule.Name)
		got, err := Is(value, rule.Name)
		require.NoError(t, err, rule.Name)
		require.True(t, got, rule.Name)
	}
	got, err := Is(true, " BOOLEAN ")
	require.NoError(t, err)
	require.True(t, got)
	_, err = Is(true, "unknown")
	require.Error(t, err)
	_, err = Is(true, "boolean", "extra")
	require.Error(t, err)
	got, err = Is("AFG", "iso_country_3")
	require.NoError(t, err)
	require.True(t, got)
	got, err = Is("afg", "iso_country_3")
	require.NoError(t, err)
	require.True(t, got)
}

func TestPredicateEdges(t *testing.T) {
	for _, tc := range []struct {
		rule  string
		value any
		want  bool
	}{
		{"null", 1, false}, {"boolean", 1, false}, {"number", math.NaN(), false}, {"number", math.Inf(1), false}, {"integer", 1.2, false}, {"integer", float64(1 << 63), false}, {"string", 1, false},
		{"array", map[string]any{}, false}, {"object", []any{}, false}, {"json", func() {}, false}, {"base64", "!", false}, {"data_uri", "data:,", false}, {"email", "bad", false}, {"url", "relative", false}, {"request_url", "ftp://example.com", false}, {"uri", "relative", false}, {"hex", "abc", false}, {"uuid", "bad", false}, {"uuid_v4", "550e8400-e29b-11d4-a716-446655440000", false}, {"mongo_id", "bad", false}, {"imei", "123", false}, {"imsi", "123", false}, {"ip", "bad", false}, {"ip_v4", "2001:db8::1", false}, {"ip_v6", "10.0.0.1", false}, {"cidr", "bad", false}, {"dns", "bad host", false}, {"hostname", "bad host", false}, {"mac", "bad", false}, {"port", int64(-1), false}, {"port", "443", true}, {"port", "bad", false}, {"rfc3339", "bad", false}, {"rfc3339_without_zone", "2026-08-30T12:00:00Z", false}, {"semver", "1", false}, {"iso_country_2", "ZZ", false}, {"iso_country_3", "ZZZ", false}, {"iso_currency", "XXX", true}, {"latitude", 91, false}, {"longitude", -181, false}, {"hash", "bad", false}, {"crc32", "bad", false}, {"regex_syntax", "[", false}, {"regex", "[", false}, {"printable_ascii", "line\n", false}, {"unicode_letter", "", false}, {"unicode_letter", "abc1", false}, {"unicode_digit", "", false}, {"unicode_digit", "12a", false}, {"color", "no", false}, {"phone", "x", false},
	} {
		got, err := Is(tc.value, tc.rule)
		require.NoError(t, err, tc.rule)
		require.Equal(t, tc.want, got, tc.rule)
	}
	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1)} {
		require.True(t, isNumber(value))
		require.True(t, isInteger(value))
		_, ok := floatValue(value)
		require.True(t, ok)
	}
	for _, value := range []any{math.NaN(), math.Inf(1), "x", nil} {
		require.False(t, isNumber(value))
		_, ok := floatValue(value)
		require.False(t, ok)
	}
	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1), " 443 "} {
		port, ok := portValue(value)
		require.True(t, ok)
		if _, isString := value.(string); isString {
			require.Equal(t, 443, port)
		} else {
			require.Equal(t, 1, port)
		}
	}
	for _, value := range []any{int64(-1), float64(65536), math.NaN(), math.Inf(1), true, "bad"} {
		_, ok := portValue(value)
		require.False(t, ok)
	}
	require.False(t, kind(map[int]int{}, 'o'))
	require.Equal(t, byte('a'), reflectKind([1]int{}))
	require.Equal(t, byte('o'), reflectKind(map[string]int{}))
	require.Equal(t, byte(0), reflectKind(1))
	require.True(t, validHostname("example.com."))
	require.False(t, validHostname(strings.Repeat("a", 254)))
	require.True(t, validJSON(jsontext.Value(`{"a":1}`)))
	require.False(t, validJSON(jsontext.Value(`{`)))
	require.True(t, finiteInteger(float64(math.MinInt64)))
	require.False(t, finiteInteger(float64(uint64(1)<<63)))
	for _, rule := range []string{"base64", "data_uri", "email", "url", "request_url", "uri", "hex", "uuid", "uuid_v1", "uuid_v3", "uuid_v4", "uuid_v5", "mongo_id", "imei", "imsi", "ip", "ip_v4", "ip_v6", "cidr", "dns", "hostname", "mac", "rfc3339", "rfc3339_without_zone", "semver", "iso_country_2", "iso_country_3", "iso_currency", "hash", "crc32", "regex_syntax", "regex", "printable_ascii", "unicode_letter", "unicode_digit", "color", "phone"} {
		got, err := Is(1, rule)
		require.NoError(t, err, rule)
		require.False(t, got, rule)
	}
}

func TestOptions(t *testing.T) {
	program, err := exprlib.Compile(`is("x", "string")`, Options()...)
	require.NoError(t, err)
	value, err := exprlib.Run(program, nil)
	require.NoError(t, err)
	require.Equal(t, true, value)
	for _, source := range []string{`is()`, `is("x")`, `is("x", 1)`, `is("x", "string", 1)`, `is("x", "unknown")`} {
		program, err := exprlib.Compile(source, Options()...)
		if err == nil {
			_, err = exprlib.Run(program, nil)
		}
		require.Error(t, err, source)
	}
}
