package emailscraper

import (
	"context"
	"fmt"
)

// Scrape is responsible for main scraping logic.
func (s *Scraper) Scrape(url string) ([]string, error) {
	return s.ScrapeWithContext(context.Background(), url)
}

// ScrapeWithContext is responsible for main scraping logic with context support.
func (s *Scraper) ScrapeWithContext(ctx context.Context, url string) ([]string, error) {
	url = getWebsite(url, true)

	// Reset emails set for new scrape
	s.emailsSet.reset()

	if !s.cfg.FollowExternalLinks {
		allowedDomains, err := prepareAllowedDomain(url)
		if err != nil {
			return nil, err
		}

		s.collector.AllowedDomains = allowedDomains
	}

	// Channel to signal completion
	done := make(chan struct{})

	go func() {
		// Start the scrape
		err := s.collector.Visit(url)
		if err != nil {
			s.log("error while visiting secure domain: ", url, err.Error())
		}

		s.collector.Wait() // Wait for concurrent scrapes to finish

		if len(s.emailsSet.toSlice()) == 0 {
			// Start the scrape on insecure url
			err := s.collector.Visit(getWebsite(url, false))
			if err != nil {
				s.log("error while visiting insecure domain: ", err.Error())
			}

			s.collector.Wait() // Wait for concurrent scrapes to finish
		}

		close(done)
	}()

	select {
	case <-ctx.Done():
		return s.emailsSet.toSlice(), fmt.Errorf("scraping canceled: %w", ctx.Err())
	case <-done:
		return s.emailsSet.toSlice(), nil
	}
}

func getWebsite(url string, secure bool) string {
	url = trimProtocol(url)

	if secure {
		return "https://" + url
	}

	return "http://" + url
}
