package emailscraper

import (
	"errors"
	"log"
	"os"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

// Scraper config.
type Scraper struct {
	cfg Config

	collector *colly.Collector
}

// Config for the scraper.
type Config struct {
	MaxDepth int
	Timeout  int

	Recursively         bool
	Async               bool
	EnableJavascript    bool
	FollowExternalLinks bool
	Debug               bool
}

// DefaultConfig defines default config with sane defaults for most use cases.
func DefaultConfig() Config {
	// nolint:gomnd // allow for default config
	return Config{
		MaxDepth:            3,
		Timeout:             5,
		Recursively:         true,
		Async:               true,
		EnableJavascript:    true,
		FollowExternalLinks: false,
		Debug:               false,
	}
}

// New initiates new scraper entity.
func New(cfg Config) *Scraper {
	// Initiate colly
	c := colly.NewCollector(
		// nolint:lll // allow long line for user agent
		colly.UserAgent("Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"),
	)

	c.Async = cfg.Async
	c.MaxDepth = cfg.MaxDepth

	if cfg.Debug {
		c.SetDebugger(&debug.LogDebugger{
			Output: os.Stderr,
			Prefix: "",
			Flag:   log.LstdFlags,
		})
	}

	s := Scraper{
		cfg:       cfg,
		collector: c,
	}

	if cfg.EnableJavascript {
		s.collector.OnResponse(func(response *colly.Response) {
			if err := initiateScrapingFromChrome(response, cfg.Timeout); err != nil {
				s.log(err)

				return
			}
		})
	}

	if cfg.Recursively {
		// Find and visit all links
		s.collector.OnHTML("a[href]", func(el *colly.HTMLElement) {
			s.log("visiting: ", el.Attr("href"))
			if err := el.Request.Visit(el.Attr("href")); err != nil {
				// Ignore already visited error, this appears too often
				if !errors.Is(err, colly.ErrAlreadyVisited) {
					s.log("error while linking: ", err.Error())
				}
			}
		})
	}

	return &s
}

func (s *Scraper) log(v ...interface{}) {
	if s.cfg.Debug {
		log.Println(v...)
	}
}
