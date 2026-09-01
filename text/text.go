// Package text provides deterministic Unicode and regular-expression helpers.
package text

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode"

	exprlib "github.com/expr-lang/expr"
	"golang.org/x/text/unicode/norm"
)

const maxPatternBytes = 8 << 10
const regexCacheSize = 128

var regexCache struct {
	sync.RWMutex
	values map[string]*regexp.Regexp
}

// RegexFind returns the first regular-expression match and its capture groups.
func RegexFind(value, pattern string) ([]string, error) {
	re, err := compile(pattern)
	if err != nil {
		return nil, err
	}
	match := re.FindStringSubmatch(value)
	if match == nil {
		return nil, nil
	}
	return match, nil
}

// RegexFindAll returns every regular-expression match and its capture groups.
func RegexFindAll(value, pattern string) ([][]string, error) {
	re, err := compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.FindAllStringSubmatch(value, -1), nil
}

// RegexReplace replaces regular-expression matches using Go replacement syntax.
func RegexReplace(value, pattern, replacement string) (string, error) {
	re, err := compile(pattern)
	if err != nil {
		return "", err
	}
	return re.ReplaceAllString(value, replacement), nil
}

// NormalizeUnicode applies the requested NFC, NFD, NFKC, or NFKD form.
func NormalizeUnicode(value, form string) (string, error) {
	var transformer norm.Form
	switch strings.ToLower(form) {
	case "", "nfc":
		transformer = norm.NFC
	case "nfd":
		transformer = norm.NFD
	case "nfkc":
		transformer = norm.NFKC
	case "nfkd":
		transformer = norm.NFKD
	default:
		return "", fmt.Errorf("normalizeUnicode: form must be nfc, nfd, nfkc, or nfkd")
	}
	return transformer.String(value), nil
}

// RemoveDiacritics removes combining marks and returns NFC-normalized text.
func RemoveDiacritics(value string) (string, error) {
	decomposed := norm.NFD.String(value)
	var out strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		out.WriteRune(r)
	}
	return norm.NFC.String(out.String()), nil
}

// Options returns the Expr registrations for text helpers.
func Options() []exprlib.Option {
	return []exprlib.Option{
		fn("regexFind", regexFindFunc),
		fn("regexFindAll", regexFindAllFunc),
		fn("regexReplace", regexReplaceFunc),
		fn("normalizeUnicode", normalizeUnicodeFunc),
		fn("removeDiacritics", removeDiacriticsFunc),
	}
}

func regexFindFunc(args ...any) (any, error) {
	value, pattern, err := regexArgs(args, "regexFind", 2)
	if err != nil {
		return nil, err
	}
	return RegexFind(value, pattern)
}

func regexFindAllFunc(args ...any) (any, error) {
	value, pattern, err := regexArgs(args, "regexFindAll", 2)
	if err != nil {
		return nil, err
	}
	return RegexFindAll(value, pattern)
}

func regexReplaceFunc(args ...any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("regexReplace: expected three arguments")
	}
	value, pattern, err := regexArgs(args[:2], "regexReplace", 2)
	if err != nil {
		return nil, err
	}
	replacement, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("regexReplace: replacement must be a string")
	}
	return RegexReplace(value, pattern, replacement)
}

func normalizeUnicodeFunc(args ...any) (any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("normalizeUnicode: expected one or two arguments")
	}
	value, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("normalizeUnicode: value must be a string")
	}
	form := "nfc"
	if len(args) == 2 {
		form, ok = args[1].(string)
		if !ok {
			return nil, fmt.Errorf("normalizeUnicode: form must be a string")
		}
	}
	return NormalizeUnicode(value, form)
}

func removeDiacriticsFunc(args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("removeDiacritics: expected one argument")
	}
	value, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("removeDiacritics: value must be a string")
	}
	return RemoveDiacritics(value)
}

func regexArgs(args []any, name string, want int) (string, string, error) {
	if len(args) != want {
		return "", "", fmt.Errorf("%s: expected two arguments", name)
	}
	value, ok := args[0].(string)
	if !ok {
		return "", "", fmt.Errorf("%s: value must be a string", name)
	}
	pattern, ok := args[1].(string)
	if !ok {
		return "", "", fmt.Errorf("%s: pattern must be a string", name)
	}
	return value, pattern, nil
}

func fn(name string, call func(...any) (any, error)) exprlib.Option {
	return exprlib.Function(name, call, new(func(...any) any))
}

func compile(pattern string) (*regexp.Regexp, error) {
	if len(pattern) > maxPatternBytes {
		return nil, fmt.Errorf("regex: pattern exceeds %d bytes", maxPatternBytes)
	}
	regexCache.RLock()
	re := regexCache.values[pattern]
	regexCache.RUnlock()
	if re != nil {
		return re, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex: %w", err)
	}
	regexCache.Lock()
	if regexCache.values == nil {
		regexCache.values = make(map[string]*regexp.Regexp, regexCacheSize)
	}
	if cached := regexCache.values[pattern]; cached != nil {
		regexCache.Unlock()
		return cached, nil
	}
	if len(regexCache.values) >= regexCacheSize {
		for key := range regexCache.values {
			delete(regexCache.values, key)
			break
		}
	}
	regexCache.values[pattern] = re
	regexCache.Unlock()
	return re, nil
}
