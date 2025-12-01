//nolint:testpackage // need access to internal functions
package emailscraper

import (
	"sync"
	"testing"
)

func TestIsValidEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"empty string", "", false},
		{"no TLD", "user@domain", false},
		{"single char TLD", "user@domain.c", false},
		{"image extension png", "user@domain.png", false},
		{"image extension jpg", "user@domain.jpg", false},
		{"image extension gif", "user@domain.gif", false},
		{"numeric TLD", "user@domain.123", false},
		{"valid simple", "user@example.com", true},
		{"valid subdomain", "user@mail.example.com", true},
		{"valid multi-part TLD", "user@example.co.uk", true},
		{"valid with dots", "first.last@example.com", true},
		{"valid with dash", "user-name@example.com", true},
		{"valid with underscore", "user_name@example.com", true},
		{"valid org TLD", "contact@example.org", true},
		{"valid net TLD", "info@example.net", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isValidEmail(testCase.email); got != testCase.expected {
				t.Errorf("isValidEmail(%q) = %v, want %v", testCase.email, got, testCase.expected)
			}
		})
	}
}

func TestDecodeCloudflareEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		encoded  string
		expected string
	}{
		{"empty", "", ""},
		{"single char", "a", ""},
		{"just key", "00", ""},
		{"invalid hex in key", "zz", ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := decodeCloudflareEmail(testCase.encoded); got != testCase.expected {
				t.Errorf("decodeCloudflareEmail(%q) = %q, want %q", testCase.encoded, got, testCase.expected)
			}
		})
	}
}

//nolint:cyclop,funlen // test function with multiple subtests
func TestParseEmails(t *testing.T) {
	t.Parallel()

	t.Run("standard email", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("Contact us at test@example.com for info"))

		result := emailSet.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 email, got %d", len(result))
		}

		if len(result) > 0 && result[0] != "test@example.com" {
			t.Errorf("expected test@example.com, got %s", result[0])
		}
	})

	t.Run("multiple emails", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("Contact test@example.com or support@example.org"))

		result := emailSet.toSlice()
		if len(result) != 2 {
			t.Errorf("expected 2 emails, got %d", len(result))
		}
	})

	t.Run("obfuscated email with brackets", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("Email: user[AT]domain.com"))

		result := emailSet.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 obfuscated email, got %d", len(result))
		}
	})

	t.Run("obfuscated email with parentheses", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("Email: user(at)domain.com"))

		result := emailSet.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 obfuscated email, got %d", len(result))
		}
	})

	t.Run("obfuscated email with spaces", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("Email: user AT domain.com"))

		result := emailSet.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 obfuscated email, got %d", len(result))
		}
	})

	t.Run("duplicate handling", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("test@example.com and test@example.com and test@example.com"))

		result := emailSet.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 unique email, got %d", len(result))
		}
	})

	t.Run("no emails in text", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("This is just some regular text without any emails"))

		result := emailSet.toSlice()
		if len(result) != 0 {
			t.Errorf("expected 0 emails, got %d", len(result))
		}
	})

	t.Run("invalid email filtered", func(t *testing.T) {
		t.Parallel()

		emailSet := &emails{} //nolint:exhaustruct // zero value is valid
		emailSet.parseEmails([]byte("image@file.png and real@example.com"))

		result := emailSet.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 valid email (png filtered), got %d", len(result))
		}
	})
}

func TestEmailsReset(t *testing.T) {
	t.Parallel()

	emailSet := &emails{} //nolint:exhaustruct // zero value is valid
	emailSet.add("test@example.com")

	if len(emailSet.toSlice()) != 1 {
		t.Errorf("expected 1 email before reset, got %d", len(emailSet.toSlice()))
	}

	emailSet.reset()

	if len(emailSet.toSlice()) != 0 {
		t.Errorf("expected 0 emails after reset, got %d", len(emailSet.toSlice()))
	}
}

func TestEmailsThreadSafety(t *testing.T) {
	t.Parallel()

	emailSet := &emails{} //nolint:exhaustruct // zero value is valid
	done := make(chan bool)
	numGoroutines := 100

	for range numGoroutines {
		go func() {
			emailSet.add("test@example.com")

			done <- true
		}()
	}

	for range numGoroutines {
		<-done
	}

	result := emailSet.toSlice()
	if len(result) != 1 {
		t.Errorf("expected 1 email after concurrent adds, got %d", len(result))
	}
}

func TestEmailsThreadSafetyMultipleEmails(t *testing.T) {
	t.Parallel()

	emailSet := &emails{} //nolint:exhaustruct // zero value is valid

	var waitGroup sync.WaitGroup

	emailAddrs := []string{
		"test1@example.com",
		"test2@example.com",
		"test3@example.com",
		"test4@example.com",
		"test5@example.com",
	}

	// Add each email 10 times concurrently
	for _, email := range emailAddrs {
		for range 10 {
			waitGroup.Add(1)

			go func(addr string) {
				defer waitGroup.Done()

				emailSet.add(addr)
			}(email)
		}
	}

	waitGroup.Wait()

	result := emailSet.toSlice()
	if len(result) != len(emailAddrs) {
		t.Errorf("expected %d unique emails, got %d", len(emailAddrs), len(result))
	}
}

func TestEmailsToSlice(t *testing.T) {
	t.Parallel()

	emailSet := &emails{} //nolint:exhaustruct // zero value is valid
	emailSet.add("a@example.com")
	emailSet.add("b@example.com")
	emailSet.add("c@example.com")

	result := emailSet.toSlice()
	if len(result) != 3 {
		t.Errorf("expected 3 emails, got %d", len(result))
	}

	// Verify all emails are present (order may vary due to map)
	resultSet := make(map[string]bool)
	for _, email := range result {
		resultSet[email] = true
	}

	for _, expected := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		if !resultSet[expected] {
			t.Errorf("expected email %s not found in result", expected)
		}
	}
}
