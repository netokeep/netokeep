package rules

import (
	"net"
	"path"
	"strings"
)

// RuleMatcher decides whether a host or IP should be allowed.
//
// Matching order:
// 1. Any deny rule matches -> false
// 2. Any allow rule matches -> true
// 3. Fallback to DefaultAllow
type RuleMatcher struct {
	DefaultAllow bool
	Allow        []string
	Deny         []string
}

// NewRuleMatcher creates a matcher with the provided default policy and rule lists.
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
// - a plain hostname, such as "localhost"
// - an IP address, such as "127.0.0.1"
// - a host with port, such as "127.0.0.1:8080" or "[::1]:8080"
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

	if strings.Contains(pattern, "*") {
		ok, err := path.Match(pattern, host)
		return err == nil && ok
	}

	if _, cidr, err := net.ParseCIDR(pattern); err == nil {
		ip := net.ParseIP(host)
		return ip != nil && cidr.Contains(ip)
	}

	if ip := net.ParseIP(pattern); ip != nil {
		hostIP := net.ParseIP(host)
		return hostIP != nil && ip.Equal(hostIP)
	}

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
