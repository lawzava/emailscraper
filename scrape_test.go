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

	emails, err := s.Scrape("https://lawzava.com")
	if err != nil {
		t.Fatal(err)
	}

	if len(emails) == 0 {
		t.Error("no emails")
		t.Fail()
	}
}
