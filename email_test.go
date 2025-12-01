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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isValidEmail(tt.email); got != tt.expected {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.expected)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := decodeCloudflareEmail(tt.encoded); got != tt.expected {
				t.Errorf("decodeCloudflareEmail(%q) = %q, want %q", tt.encoded, got, tt.expected)
			}
		})
	}
}

func TestParseEmails(t *testing.T) {
	t.Parallel()

	t.Run("standard email", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("Contact us at test@example.com for info"))

		result := e.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 email, got %d", len(result))
		}

		if len(result) > 0 && result[0] != "test@example.com" {
			t.Errorf("expected test@example.com, got %s", result[0])
		}
	})

	t.Run("multiple emails", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("Contact test@example.com or support@example.org"))

		result := e.toSlice()
		if len(result) != 2 {
			t.Errorf("expected 2 emails, got %d", len(result))
		}
	})

	t.Run("obfuscated email with brackets", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("Email: user[AT]domain.com"))

		result := e.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 obfuscated email, got %d", len(result))
		}
	})

	t.Run("obfuscated email with parentheses", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("Email: user(at)domain.com"))

		result := e.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 obfuscated email, got %d", len(result))
		}
	})

	t.Run("obfuscated email with spaces", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("Email: user AT domain.com"))

		result := e.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 obfuscated email, got %d", len(result))
		}
	})

	t.Run("duplicate handling", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("test@example.com and test@example.com and test@example.com"))

		result := e.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 unique email, got %d", len(result))
		}
	})

	t.Run("no emails in text", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("This is just some regular text without any emails"))

		result := e.toSlice()
		if len(result) != 0 {
			t.Errorf("expected 0 emails, got %d", len(result))
		}
	})

	t.Run("invalid email filtered", func(t *testing.T) {
		t.Parallel()

		e := &emails{}
		e.parseEmails([]byte("image@file.png and real@example.com"))

		result := e.toSlice()
		if len(result) != 1 {
			t.Errorf("expected 1 valid email (png filtered), got %d", len(result))
		}
	})
}

func TestEmailsReset(t *testing.T) {
	t.Parallel()

	e := &emails{}
	e.add("test@example.com")

	if len(e.toSlice()) != 1 {
		t.Errorf("expected 1 email before reset, got %d", len(e.toSlice()))
	}

	e.reset()

	if len(e.toSlice()) != 0 {
		t.Errorf("expected 0 emails after reset, got %d", len(e.toSlice()))
	}
}

func TestEmailsThreadSafety(t *testing.T) {
	t.Parallel()

	e := &emails{}
	done := make(chan bool)
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		go func() {
			e.add("test@example.com")
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	result := e.toSlice()
	if len(result) != 1 {
		t.Errorf("expected 1 email after concurrent adds, got %d", len(result))
	}
}

func TestEmailsThreadSafetyMultipleEmails(t *testing.T) {
	t.Parallel()

	e := &emails{}
	var wg sync.WaitGroup

	emails := []string{
		"test1@example.com",
		"test2@example.com",
		"test3@example.com",
		"test4@example.com",
		"test5@example.com",
	}

	// Add each email 10 times concurrently
	for _, email := range emails {
		for i := 0; i < 10; i++ {
			wg.Add(1)

			go func(em string) {
				defer wg.Done()
				e.add(em)
			}(email)
		}
	}

	wg.Wait()

	result := e.toSlice()
	if len(result) != len(emails) {
		t.Errorf("expected %d unique emails, got %d", len(emails), len(result))
	}
}

func TestEmailsToSlice(t *testing.T) {
	t.Parallel()

	e := &emails{}
	e.add("a@example.com")
	e.add("b@example.com")
	e.add("c@example.com")

	result := e.toSlice()
	if len(result) != 3 {
		t.Errorf("expected 3 emails, got %d", len(result))
	}

	// Verify all emails are present (order may vary due to map)
	emailSet := make(map[string]bool)
	for _, email := range result {
		emailSet[email] = true
	}

	for _, expected := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		if !emailSet[expected] {
			t.Errorf("expected email %s not found in result", expected)
		}
	}
}
