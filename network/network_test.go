package network

import (
	"testing"

	exprlib "github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkFunctions(t *testing.T) {
	got, err := NormalizeIP(" 192.168.1.1 ")
	require.NoError(t, err)
	require.Equal(t, "192.168.1.1", got)
	got, err = NormalizeIP("2001:DB8::1")
	require.NoError(t, err)
	require.Equal(t, "2001:db8::1", got)
	for _, value := range []string{"", "not-an-ip"} {
		_, err := NormalizeIP(value)
		require.Error(t, err)
	}
	inside, err := InCIDR("10.1.2.3", "10.0.0.0/8")
	require.NoError(t, err)
	require.True(t, inside)
	inside, err = InCIDR("192.168.1.1", "10.0.0.0/8")
	require.NoError(t, err)
	require.False(t, inside)
	for _, tc := range [][2]string{{"bad", "10.0.0.0/8"}, {"10.0.0.1", "bad"}} {
		_, err := InCIDR(tc[0], tc[1])
		require.Error(t, err)
	}
	got, err = NormalizeHostname(" Example.COM. ")
	require.NoError(t, err)
	require.Equal(t, "example.com", got)
	got, err = NormalizeHostname("127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", got)
	got, err = NormalizeHostname("bücher.de")
	require.NoError(t, err)
	require.Equal(t, "xn--bcher-kva.de", got)
	for _, value := range []string{"", "bad host"} {
		_, err := NormalizeHostname(value)
		require.Error(t, err)
	}
	got, err = NormalizeURL("HTTPS://Example.COM:443/a?b=2&a=1#x")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/a?b=2&a=1#x", got)
	got, err = NormalizeURL("HTTP://user:pass@Example.COM:8080/a")
	require.NoError(t, err)
	require.Equal(t, "http://user:pass@example.com:8080/a", got)
	got, err = NormalizeURL("http://[2001:DB8::1]:80/a")
	require.NoError(t, err)
	require.Equal(t, "http://[2001:db8::1]/a", got)
	for _, value := range []string{"", "/relative", "http://"} {
		_, err := NormalizeURL(value)
		require.Error(t, err)
	}
}

func TestOptions(t *testing.T) {
	for _, source := range []string{
		`normalizeIP("127.0.0.1")`, `inCIDR("10.1.2.3", "10.0.0.0/8")`, `normalizeURL("https://example.com:443")`, `normalizeHostname("Example.COM.")`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		require.NoError(t, err, source)
		_, err = exprlib.Run(program, nil)
		require.NoError(t, err, source)
	}
	for _, source := range []string{
		`normalizeIP()`, `normalizeIP(1)`, `inCIDR()`, `inCIDR(1, "x")`, `inCIDR("x", 1)`,
		`normalizeURL()`, `normalizeURL(1)`, `normalizeHostname()`, `normalizeHostname(1)`,
	} {
		program, err := exprlib.Compile(source, Options()...)
		if err == nil {
			_, err = exprlib.Run(program, nil)
		}
		require.Error(t, err, source)
	}
}

func TestNetworkInvariants(t *testing.T) {
	for _, value := range []string{"::ffff:192.168.1.1", "2001:0DB8::1", " 10.0.0.1 "} {
		normalized, err := NormalizeIP(value)
		require.NoError(t, err)
		again, err := NormalizeIP(normalized)
		require.NoError(t, err)
		assert.Equal(t, normalized, again)
	}

	for _, value := range []string{" Example.COM. ", "bücher.de", "127.0.0.1"} {
		normalized, err := NormalizeHostname(value)
		require.NoError(t, err)
		again, err := NormalizeHostname(normalized)
		require.NoError(t, err)
		assert.Equal(t, normalized, again)
	}

	for _, value := range []string{"HTTPS://Example.COM:443/a?b=2&a=1", "http://example.com:8080/path"} {
		normalized, err := NormalizeURL(value)
		require.NoError(t, err)
		again, err := NormalizeURL(normalized)
		require.NoError(t, err)
		assert.Equal(t, normalized, again)
	}

	for _, tc := range []struct {
		ip, cidr string
		want     bool
	}{
		{ip: "10.0.0.1", cidr: "10.0.0.0/8", want: true},
		{ip: "10.0.0.1", cidr: "192.168.0.0/16", want: false},
		{ip: "2001:db8::1", cidr: "2001:db8::/32", want: true},
	} {
		got, err := InCIDR(tc.ip, tc.cidr)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}
