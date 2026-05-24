package rules

import (
	"net"
	"net/netip"
	"strings"
)

// RuleMatcher decides whether a host or IP should be allowed.
//
// Matching order:
// 1. Any deny rule matches -> false
// 2. Any allow rule matches -> true
// 3. Fallback to DefaultAllow
//
// Rule patterns (compatible with xproxy AddFromString):
//   - "example.com"        exact hostname match
//   - "1.2.3.4"            exact IP match
//   - "10.0.0.0/8"         CIDR match
//   - "*.example.com"      zone match: example.com and all its subdomains
type RuleMatcher struct {
	DefaultAllow bool
	Allow        []string
	Deny         []string
}

func NewRuleMatcher(defaultAllow bool, allow []string, deny []string) *RuleMatcher {
	return &RuleMatcher{
		DefaultAllow: defaultAllow,
		Allow:        append([]string(nil), allow...),
		Deny:         append([]string(nil), deny...),
	}
}

// Match reports whether the provided host should be allowed.
//
// The input may be:
//   - a plain hostname, such as "localhost"
//   - an IP address, such as "127.0.0.1"
//   - a host with port, such as "127.0.0.1:8080" or "[::1]:8080"
func (r *RuleMatcher) Match(host string) bool {
	if r == nil {
		return false
	}

	normalized := normalizeHost(host)

	if matchesAny(normalized, r.Deny) {
		return false
	}
	if matchesAny(normalized, r.Allow) {
		return true
	}
	return r.DefaultAllow
}

func matchesAny(host string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPattern(host, pattern) {
			return true
		}
	}
	return false
}

func matchPattern(host, pattern string) bool {
	pattern = strings.TrimSpace(strings.ToLower(pattern))
	if pattern == "" {
		return false
	}

	host = strings.ToLower(strings.TrimSpace(host))

	// CIDR: 10.0.0.0/8, 2001:db8::/32
	if prefix, err := netip.ParsePrefix(pattern); err == nil {
		addr, err := netip.ParseAddr(host)
		return err == nil && prefix.Contains(addr)
	}

	// Exact IP: 127.0.0.1, ::1
	if addr, err := netip.ParseAddr(pattern); err == nil {
		hostAddr, err := netip.ParseAddr(host)
		return err == nil && addr == hostAddr
	}

	// Zone match: "*.example.com" matches "example.com" and all subdomains.
	// Normalise: strip the "*" prefix, then match by suffix.
	if strings.HasPrefix(pattern, "*.") {
		return host == pattern[2:] || strings.HasSuffix(host, pattern[1:])
	}

	// Exact hostname
	return host == pattern
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return host
	}

	if strings.HasPrefix(host, "[") {
		if h, _, err := net.SplitHostPort(host); err == nil {
			return h
		}
	}

	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}

	return strings.Trim(host, "[]")
}
