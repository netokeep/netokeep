package rules

import "testing"

func TestRuleMatcherMatch(t *testing.T) {
	tests := []struct {
		name    string
		matcher *RuleMatcher
		host    string
		want    bool
	}{
		{
			name:    "deny takes precedence over allow",
			matcher: NewRuleMatcher(true, []string{"localhost"}, []string{"localhost"}),
			host:    "localhost",
			want:    false,
		},
		{
			name:    "allow exact ipv4",
			matcher: NewRuleMatcher(false, []string{"127.0.0.1"}, nil),
			host:    "127.0.0.1",
			want:    true,
		},
		{
			name:    "allow exact ipv6",
			matcher: NewRuleMatcher(false, []string{"::1"}, nil),
			host:    "::1",
			want:    true,
		},
		{
			name:    "allow ipv6 with port",
			matcher: NewRuleMatcher(false, []string{"::1"}, nil),
			host:    "[::1]:8080",
			want:    true,
		},
		{
			name:    "allow ipv4 cidr",
			matcher: NewRuleMatcher(false, []string{"10.0.0.0/8"}, nil),
			host:    "10.1.2.3",
			want:    true,
		},
		{
			name:    "allow ipv6 cidr",
			matcher: NewRuleMatcher(false, []string{"2001:db8::/32"}, nil),
			host:    "2001:db8::1",
			want:    true,
		},
		{
			name:    "allow suffix match subdomain",
			matcher: NewRuleMatcher(false, []string{"*.local"}, nil),
			host:    "demo.local",
			want:    true,
		},
		{
			name:    "allow suffix match base domain",
			matcher: NewRuleMatcher(false, []string{"*.local"}, nil),
			host:    "local",
			want:    true,
		},
		{
			name:    "deny suffix match github domain",
			matcher: NewRuleMatcher(true, []string{"*.local"}, []string{"*.github.com"}),
			host:    "api.github.com",
			want:    false,
		},
		{
			name:    "fallback to default allow",
			matcher: NewRuleMatcher(true, nil, nil),
			host:    "example.com",
			want:    true,
		},
		{
			name:    "fallback to default deny",
			matcher: NewRuleMatcher(false, nil, nil),
			host:    "example.com",
			want:    false,
		},
		{
			name:    "nil matcher denies",
			matcher: nil,
			host:    "example.com",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.Match(tt.host); got != tt.want {
				t.Fatalf("Match(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
