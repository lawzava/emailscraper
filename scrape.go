package emailscraper

import (
	"github.com/gocolly/colly"
)

// Scrape is responsible for main scraping logic.
func (s *Scraper) Scrape(url string) ([]string, error) {
	url = getWebsite(url, true)
	e := emails{}

	c := s.collector

	if !s.cfg.FollowExternalLinks {
		allowedDomains, err := prepareAllowedDomain(url)
		if err != nil {
			return nil, err
		}

		c.AllowedDomains = allowedDomains
	}

	// Parse emails on each downloaded page
	c.OnScraped(func(response *colly.Response) {
		e.parseEmails(response.Body)
	})

	// cloudflare encoded email support
	c.OnHTML("span[data-cfemail]", func(el *colly.HTMLElement) {
		e.parseCloudflareEmail(el.Attr("data-cfemail"))
	})

	// Start the scrape
	if err := c.Visit(url); err != nil {
		s.log("error while visiting secure domain: ", url, err.Error())
	}

	c.Wait() // Wait for concurrent scrapes to finish

	if e.emails == nil || len(e.emails) == 0 {
		// Start the scrape on insecure url
		if err := c.Visit(getWebsite(url, false)); err != nil {
			s.log("error while visiting insecure domain: ", err.Error())
		}

		c.Wait() // Wait for concurrent scrapes to finish
	}

	return e.emails, nil
}

func getWebsite(url string, secure bool) string {
	url = trimProtocol(url)

	if secure {
		return "https://" + url
	}

	return "http://" + url
}
