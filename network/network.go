// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

// Package network provides pure IP, CIDR, URL, and hostname normalization.
package network

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	exprlib "github.com/expr-lang/expr"
	"golang.org/x/net/idna"
)

// NormalizeIP parses an IP address and returns its canonical textual form.
func NormalizeIP(value string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return "", fmt.Errorf("normalizeIP: invalid IP address")
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String(), nil
	}
	return ip.String(), nil
}

// InCIDR reports whether ip belongs to the supplied CIDR network.
func InCIDR(ip, cidr string) (bool, error) {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false, fmt.Errorf("inCIDR: invalid IP address")
	}
	_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return false, fmt.Errorf("inCIDR: invalid CIDR: %w", err)
	}
	return network.Contains(parsed), nil
}

// NormalizeHostname lowercases and converts a hostname to ASCII lookup form.
func NormalizeHostname(value string) (string, error) {
	host := strings.TrimSpace(value)
	if host == "" {
		return "", fmt.Errorf("normalizeHostname: hostname is required")
	}
	if strings.HasSuffix(host, ".") {
		host = strings.TrimSuffix(host, ".")
	}
	if ip := net.ParseIP(host); ip != nil {
		return NormalizeIP(host)
	}
	host, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return "", fmt.Errorf("normalizeHostname: %w", err)
	}
	return strings.ToLower(host), nil
}

// NormalizeURL canonicalizes an absolute URL, including host and default ports.
func NormalizeURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("normalizeURL: absolute URL is required")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	hostname, err := NormalizeHostname(parsed.Hostname())
	if err != nil {
		return "", err
	}
	port := parsed.Port()
	if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
		port = ""
	}
	host := hostname
	if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port != "" {
		host += ":" + port
	}
	parsed.Host = host
	return parsed.String(), nil
}

// Options returns the Expr registrations for network helpers.
func Options() []exprlib.Option {
	return []exprlib.Option{
		fn("normalizeIP", func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("normalizeIP: expected one argument")
			}
			value, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("normalizeIP: value must be a string")
			}
			return NormalizeIP(value)
		}),
		fn("inCIDR", func(args ...any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("inCIDR: expected two arguments")
			}
			ip, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("inCIDR: IP must be a string")
			}
			cidr, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("inCIDR: CIDR must be a string")
			}
			return InCIDR(ip, cidr)
		}),
		fn("normalizeURL", func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("normalizeURL: expected one argument")
			}
			value, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("normalizeURL: value must be a string")
			}
			return NormalizeURL(value)
		}),
		fn("normalizeHostname", func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("normalizeHostname: expected one argument")
			}
			value, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("normalizeHostname: value must be a string")
			}
			return NormalizeHostname(value)
		}),
	}
}

func fn(name string, call func(...any) (any, error)) exprlib.Option {
	return exprlib.Function(name, call, new(func(...any) any))
}
