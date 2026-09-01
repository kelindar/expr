// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

// Package validate provides pure value predicates and a stable rule catalog.
package validate

import (
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	exprlib "github.com/expr-lang/expr"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

// Rule describes one expression validation rule.
type Rule struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Example string `json:"example"`
}

type ruleDoc struct {
	summary string
	example string
}

var ruleDocs = map[string]ruleDoc{
	"null":                 {"Matches only null values.", `is(this.deleted_at, "null")`},
	"boolean":              {"Matches boolean true or false values.", `is(this.enabled, "boolean")`},
	"number":               {"Matches finite numeric values; NaN and infinity are rejected.", `is(this.score, "number")`},
	"integer":              {"Matches integer values, including whole finite floating-point numbers.", `is(this.retries, "integer")`},
	"string":               {"Matches string values, including an empty string.", `is(this.name, "string")`},
	"array":                {"Matches arrays and slices.", `is(this.items, "array")`},
	"object":               {"Matches objects and maps whose keys are strings.", `is(this.metadata, "object")`},
	"json":                 {"Matches values that can be encoded as valid JSON.", `is(this.payload, "json")`},
	"base64":               {"Matches standard padded Base64 text.", `is(this.attachment, "base64")`},
	"data_uri":             {"Matches a non-empty data URI with a media type and payload.", `is(this.image, "data_uri")`},
	"email":                {"Matches a basic email address with a local part, @ sign, and dotted domain.", `is(this.email, "email")`},
	"url":                  {"Matches an absolute URL or URI with a scheme.", `is(this.website, "url")`},
	"request_url":          {"Matches an absolute HTTP or HTTPS request URL with a host.", `is(this.callback_url, "request_url")`},
	"uri":                  {"Matches an absolute URI with a scheme, including non-HTTP schemes.", `is(this.resource, "uri")`},
	"hex":                  {"Matches an even-length hexadecimal byte string.", `is(this.payload, "hex")`},
	"uuid":                 {"Matches a canonical hyphenated UUID.", `is(this.id, "uuid")`},
	"uuid_v1":              {"Matches a canonical version 1 UUID.", `is(this.id, "uuid_v1")`},
	"uuid_v3":              {"Matches a canonical version 3 UUID.", `is(this.id, "uuid_v3")`},
	"uuid_v4":              {"Matches a canonical version 4 UUID.", `is(this.id, "uuid_v4")`},
	"uuid_v5":              {"Matches a canonical version 5 UUID.", `is(this.id, "uuid_v5")`},
	"mongo_id":             {"Matches a 24-character hexadecimal MongoDB ObjectId.", `is(this.id, "mongo_id")`},
	"imei":                 {"Matches exactly 15 decimal digits; it does not verify the check digit.", `is(this.device_id, "imei")`},
	"imsi":                 {"Matches an IMSI-like sequence of 6 to 15 decimal digits.", `is(this.subscriber_id, "imsi")`},
	"ip":                   {"Matches either an IPv4 or IPv6 address.", `is(this.address, "ip")`},
	"ip_v4":                {"Matches an IPv4 address.", `is(this.address, "ip_v4")`},
	"ip_v6":                {"Matches an IPv6 address.", `is(this.address, "ip_v6")`},
	"cidr":                 {"Matches an IPv4 or IPv6 CIDR network.", `is(this.network, "cidr")`},
	"dns":                  {"Matches a DNS-style hostname up to 253 characters.", `is(this.domain, "dns")`},
	"hostname":             {"Matches a DNS-style hostname up to 253 characters.", `is(this.host, "hostname")`},
	"mac":                  {"Matches a MAC address accepted by Go's network parser.", `is(this.mac, "mac")`},
	"port":                 {"Matches an integer or decimal string from 0 through 65535.", `is(this.port, "port")`},
	"rfc3339":              {"Matches an RFC 3339 timestamp with a time-zone offset.", `is(this.created_at, "rfc3339")`},
	"rfc3339_without_zone": {"Matches an RFC 3339-style local timestamp without a time zone.", `is(this.local_time, "rfc3339_without_zone")`},
	"semver":               {"Matches a semantic version with major, minor, and patch numbers.", `is(this.version, "semver")`},
	"iso_country_2":        {"Matches a two-letter ISO 3166 country code, case-insensitively.", `is(this.country, "iso_country_2")`},
	"iso_country_3":        {"Matches a three-letter ISO 3166 country code, case-insensitively.", `is(this.country, "iso_country_3")`},
	"iso_currency":         {"Matches a three-letter ISO 4217 currency code, case-insensitively.", `is(this.currency, "iso_currency")`},
	"latitude":             {"Matches a finite number from -90 through 90.", `is(this.latitude, "latitude")`},
	"longitude":            {"Matches a finite number from -180 through 180.", `is(this.longitude, "longitude")`},
	"hash":                 {"Matches a hexadecimal digest of 16, 32, 40, 64, 96, or 128 characters.", `is(this.digest, "hash")`},
	"crc32":                {"Matches an 8-character hexadecimal CRC-32 checksum.", `is(this.checksum, "crc32")`},
	"regex_syntax":         {"Matches text that compiles as a valid RE2 regular expression.", `is(this.pattern, "regex_syntax")`},
	"regex":                {"Matches text that compiles as a valid RE2 regular expression.", `is(this.pattern, "regex")`},
	"printable_ascii":      {"Matches strings containing only printable ASCII characters.", `is(this.label, "printable_ascii")`},
	"unicode_letter":       {"Matches a non-empty string made entirely of Unicode letters.", `is(this.word, "unicode_letter")`},
	"unicode_digit":        {"Matches a non-empty string made entirely of Unicode decimal digits.", `is(this.code, "unicode_digit")`},
	"color":                {"Matches a 3-, 6-, or 8-digit hexadecimal color, with an optional # prefix.", `is(this.color, "color")`},
	"phone":                {"Matches a phone-like string of digits and common separators, optionally starting with +.", `is(this.phone, "phone")`},
}

var rules = map[string]func(any, []any) bool{
	"null":    func(v any, _ []any) bool { return v == nil },
	"boolean": func(v any, _ []any) bool { _, ok := v.(bool); return ok },
	"number":  func(v any, _ []any) bool { return isNumber(v) },
	"integer": func(v any, _ []any) bool { return isInteger(v) },
	"string":  func(v any, _ []any) bool { _, ok := v.(string); return ok },
	"array":   func(v any, _ []any) bool { return kind(v, 'a') },
	"object":  func(v any, _ []any) bool { return kind(v, 'o') },
	"json":    func(v any, _ []any) bool { return validJSON(v) },
	"base64": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		_, err := base64.StdEncoding.DecodeString(s)
		return err == nil
	},
	"data_uri": func(v any, _ []any) bool { s, ok := stringValue(v); return ok && dataURI.MatchString(s) },
	"email":    func(v any, _ []any) bool { s, ok := stringValue(v); return ok && email.MatchString(s) },
	"url": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		parsed, err := url.Parse(s)
		return err == nil && parsed.Scheme != ""
	},
	"request_url": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		parsed, err := url.ParseRequestURI(s)
		return err == nil && parsed.IsAbs() && parsed.Host != "" && (strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https"))
	},
	"uri": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		parsed, err := url.Parse(s)
		return err == nil && parsed.Scheme != ""
	},
	"hex": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		return ok && len(s)%2 == 0 && hexPattern.MatchString(s)
	},
	"uuid":     func(v any, _ []any) bool { s, ok := stringValue(v); return ok && uuidPattern.MatchString(s) },
	"uuid_v1":  func(v any, _ []any) bool { return uuidVersion(v, "1") },
	"uuid_v3":  func(v any, _ []any) bool { return uuidVersion(v, "3") },
	"uuid_v4":  func(v any, _ []any) bool { return uuidVersion(v, "4") },
	"uuid_v5":  func(v any, _ []any) bool { return uuidVersion(v, "5") },
	"mongo_id": func(v any, _ []any) bool { s, ok := stringValue(v); return ok && mongoID.MatchString(s) },
	"imei":     func(v any, _ []any) bool { s, ok := stringValue(v); return ok && digits.MatchString(s) && len(s) == 15 },
	"imsi": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		return ok && digits.MatchString(s) && len(s) >= 6 && len(s) <= 15
	},
	"ip": func(v any, _ []any) bool { s, ok := stringValue(v); return ok && net.ParseIP(s) != nil },
	"ip_v4": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		return ok && net.ParseIP(s) != nil && net.ParseIP(s).To4() != nil
	},
	"ip_v6": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		return ok && net.ParseIP(s) != nil && net.ParseIP(s).To4() == nil
	},
	"cidr": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		_, _, err := net.ParseCIDR(s)
		return err == nil
	},
	"dns": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		return ok && validHostname(s)
	},
	"hostname": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		return ok && validHostname(s)
	},
	"mac": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		_, err := net.ParseMAC(s)
		return err == nil
	},
	"port": func(v any, _ []any) bool { n, ok := portValue(v); return ok && n >= 0 && n <= 65535 },
	"rfc3339": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		_, err := timeParseRFC3339(s)
		return err == nil
	},
	"rfc3339_without_zone": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		if !rfc3339NoZone.MatchString(s) {
			return false
		}
		_, err := time.Parse("2006-01-02T15:04:05.999999999", s)
		return err == nil
	},
	"semver": func(v any, _ []any) bool { s, ok := stringValue(v); return ok && semver.MatchString(s) },
	"iso_country_2": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok || len(s) != 2 {
			return false
		}
		region, err := language.ParseRegion(strings.ToUpper(s))
		return err == nil && region.IsCountry()
	},
	"iso_country_3": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok || len(s) != 3 {
			return false
		}
		region, err := language.ParseRegion(strings.ToUpper(s))
		return err == nil && region.IsCountry()
	},
	"iso_currency": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		_, err := currency.ParseISO(strings.ToUpper(s))
		return err == nil
	},
	"latitude":  func(v any, _ []any) bool { n, ok := floatValue(v); return ok && n >= -90 && n <= 90 },
	"longitude": func(v any, _ []any) bool { n, ok := floatValue(v); return ok && n >= -180 && n <= 180 },
	"hash":      func(v any, _ []any) bool { s, ok := stringValue(v); return ok && hashPattern.MatchString(s) },
	"crc32":     func(v any, _ []any) bool { s, ok := stringValue(v); return ok && crc32Pattern.MatchString(s) },
	"regex_syntax": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		_, err := regexp.Compile(s)
		return err == nil
	},
	"regex": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		_, err := regexp.Compile(s)
		return err == nil
	},
	"printable_ascii": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok {
			return false
		}
		for _, r := range s {
			if r < 0x20 || r > 0x7e {
				return false
			}
		}
		return true
	},
	"unicode_letter": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok || s == "" {
			return false
		}
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return false
			}
		}
		return true
	},
	"unicode_digit": func(v any, _ []any) bool {
		s, ok := stringValue(v)
		if !ok || s == "" {
			return false
		}
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return false
			}
		}
		return true
	},
	"color": func(v any, _ []any) bool { s, ok := stringValue(v); return ok && colorPattern.MatchString(s) },
	"phone": func(v any, _ []any) bool { s, ok := stringValue(v); return ok && phonePattern.MatchString(s) },
}

var (
	dataURI       = regexp.MustCompile(`(?i)^data:[^,]+,.+$`)
	email         = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	hexPattern    = regexp.MustCompile(`^[0-9a-fA-F]+$`)
	uuidPattern   = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	mongoID       = regexp.MustCompile(`^[0-9a-fA-F]{24}$`)
	digits        = regexp.MustCompile(`^[0-9]+$`)
	hostname      = regexp.MustCompile(`^([A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`)
	rfc3339NoZone = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?$`)
	semver        = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	hashPattern   = regexp.MustCompile(`(?i)^(?:[0-9a-f]{16}|[0-9a-f]{32}|[0-9a-f]{40}|[0-9a-f]{64}|[0-9a-f]{96}|[0-9a-f]{128})$`)
	crc32Pattern  = regexp.MustCompile(`(?i)^[0-9a-f]{8}$`)
	colorPattern  = regexp.MustCompile(`(?i)^#?(?:[0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$`)
	phonePattern  = regexp.MustCompile(`^\+?[0-9][0-9 .()\-]{5,}$`)
)

// Is applies one named pure validation rule.
func Is(value any, rule string, args ...any) (bool, error) {
	rule = strings.ToLower(strings.TrimSpace(rule))
	predicate, ok := rules[rule]
	if !ok {
		return false, fmt.Errorf("validate: unknown rule %q", rule)
	}
	if len(args) != 0 {
		return false, fmt.Errorf("validate: rule %q does not accept arguments", rule)
	}
	return predicate(value, args), nil
}

// Rules returns the stable, alphabetically ordered validation catalog.
func Rules() []Rule {
	out := make([]Rule, 0, len(rules))
	for name := range rules {
		doc := ruleDocs[name]
		out = append(out, Rule{Name: name, Summary: doc.summary, Example: doc.example})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Options returns the Expr registrations for validation helpers.
func Options() []exprlib.Option {
	return []exprlib.Option{
		exprlib.Function("is", func(args ...any) (any, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("is: expected value and rule")
			}
			rule, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("is: rule must be a string")
			}
			return Is(args[0], rule, args[2:]...)
		}, new(func(...any) bool)),
	}
}

func uuidVersion(value any, version string) bool {
	s, ok := stringValue(value)
	return ok && uuidPattern.MatchString(s) && strings.EqualFold(string(s[14]), version)
}

func kind(value any, want byte) bool {
	if value == nil {
		return false
	}
	switch reflectKind(value) {
	case 'a':
		return want == 'a'
	case 'o':
		return want == 'o'
	default:
		return false
	}
}

func reflectKind(value any) byte {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return 0
	}
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		return 'a'
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			return 'o'
		}
	}
	return 0
}

func stringValue(value any) (string, bool) { out, ok := value.(string); return out, ok }

func isNumber(value any) bool {
	switch x := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return !isNaN(float64(x))
	case float64:
		return !isNaN(x)
	default:
		return false
	}
}

func isInteger(value any) bool {
	switch x := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		f := float64(x)
		return finiteInteger(f)
	case float64:
		return finiteInteger(x)
	default:
		return false
	}
}

func finiteInteger(value float64) bool {
	const limit = float64(uint64(1) << 63)
	return !isNaN(value) && value >= -limit && value < limit && value == math.Trunc(value)
}

func validJSON(value any) bool {
	if raw, ok := value.(jsontext.Value); ok {
		return raw.IsValid()
	}
	_, err := json.Marshal(value)
	return err == nil
}

func floatValue(value any) (float64, bool) {
	switch x := value.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), !isNaN(float64(x))
	case float64:
		return x, !isNaN(x)
	default:
		return 0, false
	}
}

func portValue(value any) (int, bool) {
	if s, ok := value.(string); ok {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		return n, err == nil
	}
	n, ok := floatValue(value)
	if !ok || !finiteInteger(n) || n < 0 || n > 65535 {
		return 0, false
	}
	return int(n), true
}

func validHostname(value string) bool {
	value = strings.TrimSuffix(value, ".")
	return len(value) > 0 && len(value) <= 253 && hostname.MatchString(value)
}

func timeParseRFC3339(value string) (interface{}, error) {
	return time.Parse(time.RFC3339, value)
}

func isNaN(value float64) bool { return math.IsNaN(value) || math.IsInf(value, 0) }
