package emailscraper_test

import (
	"testing"

	"github.com/lawzava/emailscraper"
)

func TestScrape(t *testing.T) {
	t.Parallel()

	cfg := emailscraper.DefaultConfig()
	cfg.Debug = true
	cfg.MaxDepth = 1

	scraper := emailscraper.New(cfg)

	testCases := []struct {
		name             string
		url              string
		mustContainEmail string
	}{
		{"cloudflare protected", "https://lawzava.com/contact/", "contact@lawzava.com"},
	}

	for _, testCase := range testCases {
		emails, err := scraper.Scrape(testCase.url)
		if err != nil {
			t.Fatal(err)
		}

		var contains bool

		for _, email := range emails {
			if email == testCase.mustContainEmail {
				contains = true

				break
			}
		}

		if !contains {
			t.Error("email missing: ", emails)
			t.Fail()
		}
	}
}
