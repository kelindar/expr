// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

// Package encoding provides stable JSON encoding, hashing, and checksums.
package encoding

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"hash/crc32"
	"net/url"
	"strings"
	"unicode/utf8"

	exprlib "github.com/expr-lang/expr"
	"github.com/kelindar/expr/internal/jcs"
	"github.com/zeebo/xxh3"
)

// CanonicalJSON returns the RFC 8785 canonical form of one JSON value.
func CanonicalJSON(raw []byte) ([]byte, error) {
	if len(raw) == 0 || !jsontext.Value(raw).IsValid() {
		return nil, fmt.Errorf("canonical json: value must be valid json")
	}
	out, err := jcs.Transform(raw)
	if err != nil {
		return nil, fmt.Errorf("canonical json: %w", err)
	}
	return out, nil
}

// Hash hashes strings as exact UTF-8 and other values as canonical JSON.
func Hash(value any, algorithm string) (string, error) {
	data, err := bytesFor(value)
	if err != nil {
		return "", err
	}
	var sum []byte
	switch strings.ToLower(algorithm) {
	case "md5":
		h := md5.Sum(data)
		sum = h[:]
	case "sha1":
		h := sha1.Sum(data)
		sum = h[:]
	case "sha256":
		h := sha256.Sum256(data)
		sum = h[:]
	case "sha384":
		h := sha512.Sum384(data)
		sum = h[:]
	case "sha512":
		h := sha512.Sum512(data)
		sum = h[:]
	case "xxh3":
		return fmt.Sprintf("%016x", xxh3.Hash(data)), nil
	default:
		return "", fmt.Errorf("hash: unsupported algorithm %q", algorithm)
	}
	return hex.EncodeToString(sum), nil
}

// Checksum computes the requested compatibility checksum.
func Checksum(value any, algorithm string) (string, error) {
	if strings.ToLower(algorithm) != "crc32" {
		return "", fmt.Errorf("checksum: algorithm must be crc32")
	}
	data, err := bytesFor(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE(data)), nil
}

// HexEncode returns lowercase hexadecimal for the UTF-8 bytes of value.
func HexEncode(value string) string { return hex.EncodeToString([]byte(value)) }

// HexDecode decodes hexadecimal text and requires valid UTF-8 output.
func HexDecode(value string) (string, error) {
	data, err := hex.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("hexDecode: %w", err)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("hexDecode: decoded bytes are not valid UTF-8")
	}
	return string(data), nil
}

// URLEncode escapes value using the explicit query or path_segment mode.
func URLEncode(value, mode string) (string, error) {
	switch mode {
	case "query":
		return url.QueryEscape(value), nil
	case "path_segment":
		return url.PathEscape(value), nil
	default:
		return "", fmt.Errorf("urlEncode: mode must be query or path_segment")
	}
}

// URLDecode unescapes value using the explicit query or path_segment mode.
func URLDecode(value, mode string) (string, error) {
	var (
		out string
		err error
	)
	switch mode {
	case "query":
		out, err = url.QueryUnescape(value)
	case "path_segment":
		out, err = url.PathUnescape(value)
	default:
		return "", fmt.Errorf("urlDecode: mode must be query or path_segment")
	}
	if err != nil {
		return "", fmt.Errorf("urlDecode: %w", err)
	}
	if !utf8.ValidString(out) {
		return "", fmt.Errorf("urlDecode: decoded bytes are not valid UTF-8")
	}
	return out, nil
}

// Options returns the Expr registrations for encoding helpers.
func Options() []exprlib.Option {
	return []exprlib.Option{
		fn("hash", hashFunc),
		fn("checksum", checksumFunc),
		fn("hexEncode", hexEncodeFunc),
		fn("hexDecode", hexDecodeFunc),
		fn("urlEncode", urlEncodeFunc),
		fn("urlDecode", urlDecodeFunc),
	}
}

func hashFunc(args ...any) (any, error) {
	value, algorithm, err := algorithmArgs(args, "hash")
	if err != nil {
		return nil, err
	}
	return Hash(value, algorithm)
}

func checksumFunc(args ...any) (any, error) {
	value, algorithm, err := algorithmArgs(args, "checksum")
	if err != nil {
		return nil, err
	}
	return Checksum(value, algorithm)
}

func hexEncodeFunc(args ...any) (any, error) {
	value, err := stringArg(args, "hexEncode")
	if err != nil {
		return nil, err
	}
	return HexEncode(value), nil
}

func hexDecodeFunc(args ...any) (any, error) {
	value, err := stringArg(args, "hexDecode")
	if err != nil {
		return nil, err
	}
	return HexDecode(value)
}

func urlEncodeFunc(args ...any) (any, error) {
	value, mode, err := modeArgs(args, "urlEncode")
	if err != nil {
		return nil, err
	}
	return URLEncode(value, mode)
}

func urlDecodeFunc(args ...any) (any, error) {
	value, mode, err := modeArgs(args, "urlDecode")
	if err != nil {
		return nil, err
	}
	return URLDecode(value, mode)
}

func algorithmArgs(args []any, name string) (any, string, error) {
	if len(args) != 2 {
		return nil, "", fmt.Errorf("%s: expected two arguments", name)
	}
	algorithm, ok := args[1].(string)
	if !ok {
		return nil, "", fmt.Errorf("%s: algorithm must be a string", name)
	}
	return args[0], algorithm, nil
}

func stringArg(args []any, name string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%s: expected one argument", name)
	}
	value, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("%s: value must be a string", name)
	}
	return value, nil
}

func modeArgs(args []any, name string) (string, string, error) {
	if len(args) != 2 {
		return "", "", fmt.Errorf("%s: expected value and mode", name)
	}
	value, ok := args[0].(string)
	if !ok {
		return "", "", fmt.Errorf("%s: value must be a string", name)
	}
	mode, ok := args[1].(string)
	if !ok {
		return "", "", fmt.Errorf("%s: mode must be a string", name)
	}
	return value, mode, nil
}

func fn(name string, call func(...any) (any, error)) exprlib.Option {
	return exprlib.Function(name, call, new(func(...any) any))
}

func bytesFor(value any) ([]byte, error) {
	if text, ok := value.(string); ok {
		return []byte(text), nil
	}
	return canonicalValue(value)
}

func canonicalValue(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("value is not representable as json: %w", err)
	}
	return CanonicalJSON(raw)
}
