package emailscraper_test

import (
	"slices"
	"testing"

	"github.com/lawzava/emailscraper"
)

func TestScrape(t *testing.T) {
	t.Parallel()

	cfg := emailscraper.DefaultConfig()
	cfg.Debug = true
	cfg.MaxDepth = 1

	testCases := []struct {
		name             string
		url              string
		mustContainEmail string
	}{
		{"cloudflare protected", "https://lawzava.com/contact/", "contact@lawzava.com"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scraper := emailscraper.New(cfg)

			emails, err := scraper.Scrape(testCase.url)
			if err != nil {
				t.Fatalf("Scrape() error: %v", err)
			}

			if !slices.Contains(emails, testCase.mustContainEmail) {
				t.Errorf("email %q missing, got: %v", testCase.mustContainEmail, emails)
			}
		})
	}
}
