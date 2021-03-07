package emailscraper_test

import (
	"testing"

	"github.com/lawzava/emailscraper"
)

func TestScrape(t *testing.T) {
	t.Parallel()

	cfg := emailscraper.DefaultConfig()
	cfg.Debug = true

	s := emailscraper.New(cfg)

	testCases := []struct {
		name             string
		url              string
		mustContainEmail string
	}{
		{"cloudflare protected", "https://lawzava.com/contact/", "law@zava.dev"},
	}

	for _, testCase := range testCases {
		emails, err := s.Scrape(testCase.url)
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
