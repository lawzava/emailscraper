package emailscraper

import (
	"testing"
)

func TestPrepareAllowedDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		url         string
		wantDomains []string
		wantErr     bool
	}{
		{
			name:        "simple domain",
			url:         "example.com",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "with www prefix",
			url:         "www.example.com",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "with https protocol",
			url:         "https://example.com",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "with http protocol",
			url:         "http://example.com",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "with path",
			url:         "example.com/path/to/page",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "with https and path",
			url:         "https://example.com/contact",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "with www and path",
			url:         "www.example.com/about",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "subdomain preserved",
			url:         "blog.example.com",
			wantDomains: []string{"blog.example.com", "www.blog.example.com"},
			wantErr:     false,
		},
		{
			name:        "with query string",
			url:         "example.com?foo=bar",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
		{
			name:        "with port number",
			url:         "example.com:8080",
			wantDomains: []string{"example.com", "www.example.com"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := prepareAllowedDomain(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareAllowedDomain(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}

			if !tt.wantErr && !equalStringSlices(got, tt.wantDomains) {
				t.Errorf("prepareAllowedDomain(%q) = %v, want %v", tt.url, got, tt.wantDomains)
			}
		})
	}
}

func TestTrimProtocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "https protocol",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "http protocol",
			url:      "http://example.com",
			expected: "example.com",
		},
		{
			name:     "no protocol",
			url:      "example.com",
			expected: "example.com",
		},
		{
			name:     "https with path",
			url:      "https://example.com/path",
			expected: "example.com/path",
		},
		{
			name:     "http with path",
			url:      "http://example.com/path/to/page",
			expected: "example.com/path/to/page",
		},
		{
			name:     "https with query",
			url:      "https://example.com?foo=bar",
			expected: "example.com?foo=bar",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "only https",
			url:      "https://",
			expected: "",
		},
		{
			name:     "only http",
			url:      "http://",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := trimProtocol(tt.url); got != tt.expected {
				t.Errorf("trimProtocol(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
