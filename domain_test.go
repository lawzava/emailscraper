//nolint:testpackage // need access to internal functions
package emailscraper

import (
	"testing"
)

//nolint:funlen // table-driven test with many cases
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := prepareAllowedDomain(testCase.url)
			if (err != nil) != testCase.wantErr {
				t.Errorf("prepareAllowedDomain(%q) error = %v, wantErr %v", testCase.url, err, testCase.wantErr)

				return
			}

			if !testCase.wantErr && !equalStringSlices(got, testCase.wantDomains) {
				t.Errorf("prepareAllowedDomain(%q) = %v, want %v", testCase.url, got, testCase.wantDomains)
			}
		})
	}
}

//nolint:funlen // table-driven test with many cases
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := trimProtocol(testCase.url); got != testCase.expected {
				t.Errorf("trimProtocol(%q) = %q, want %q", testCase.url, got, testCase.expected)
			}
		})
	}
}

func equalStringSlices(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}

	for idx := range first {
		if first[idx] != second[idx] {
			return false
		}
	}

	return true
}
