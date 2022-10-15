package emailscraper

import (
	"github.com/gocolly/colly/v2"
)

// Scrape is responsible for main scraping logic.
func (s *Scraper) Scrape(url string) ([]string, error) {
	url = getWebsite(url, true)

	var emailsSet emails

	collector := s.collector

	if !s.cfg.FollowExternalLinks {
		allowedDomains, err := prepareAllowedDomain(url)
		if err != nil {
			return nil, err
		}

		collector.AllowedDomains = allowedDomains
	}

	// Parse emails on each downloaded page
	collector.OnScraped(func(response *colly.Response) {
		emailsSet.parseEmails(response.Body)
	})

	// cloudflare encoded email support
	collector.OnHTML("span[data-cfemail]", func(el *colly.HTMLElement) {
		emailsSet.parseCloudflareEmail(el.Attr("data-cfemail"))
	})

	// Start the scrape
	if err := collector.Visit(url); err != nil {
		s.log("error while visiting secure domain: ", url, err.Error())
	}

	collector.Wait() // Wait for concurrent scrapes to finish

	if emailsSet.emails == nil || len(emailsSet.emails) == 0 {
		// Start the scrape on insecure url
		if err := collector.Visit(getWebsite(url, false)); err != nil {
			s.log("error while visiting insecure domain: ", err.Error())
		}

		collector.Wait() // Wait for concurrent scrapes to finish
	}

	return emailsSet.emails, nil
}

func getWebsite(url string, secure bool) string {
	url = trimProtocol(url)

	if secure {
		return "https://" + url
	}

	return "http://" + url
}
