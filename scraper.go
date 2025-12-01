package emailscraper

import (
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

const (
	// defaultMaxDepth is the default maximum depth for scraping.
	defaultMaxDepth = 3
	// defaultTimeoutSeconds is the default timeout in seconds for scraping operations.
	defaultTimeoutSeconds = 30
	// defaultRateLimitDelay is the default delay between requests to the same domain.
	defaultRateLimitDelay = 100 * time.Millisecond
	// defaultParallelism is the default number of concurrent requests per domain.
	defaultParallelism = 2
	// defaultMaxRetries is the default number of retry attempts for failed requests.
	defaultMaxRetries = 3
	// defaultRetryDelay is the default initial delay between retries.
	defaultRetryDelay = 1 * time.Second
	// maxRetryDelay is the maximum delay between retries.
	maxRetryDelay = 30 * time.Second
)

// Scraper config.
type Scraper struct {
	cfg Config

	collector *colly.Collector
	emailsSet *emails
}

// Config for the scraper.
type Config struct {
	MaxDepth int
	Timeout  int

	// Rate limiting
	RateLimitDelay time.Duration
	Parallelism    int

	// Retry configuration
	MaxRetries int
	RetryDelay time.Duration

	// Behavior flags
	Recursively         bool
	Async               bool
	EnableJavascript    bool
	FollowExternalLinks bool
	RespectRobotsTxt    bool
	Debug               bool
}

// DefaultConfig defines default config with sane defaults for most use cases.
func DefaultConfig() Config {
	return Config{
		MaxDepth:            defaultMaxDepth,
		Timeout:             defaultTimeoutSeconds,
		RateLimitDelay:      defaultRateLimitDelay,
		Parallelism:         defaultParallelism,
		MaxRetries:          defaultMaxRetries,
		RetryDelay:          defaultRetryDelay,
		Recursively:         true,
		Async:               true,
		EnableJavascript:    true,
		FollowExternalLinks: false,
		RespectRobotsTxt:    true,
		Debug:               false,
	}
}

// New initiates new scraper entity.
func New(cfg Config) *Scraper {
	// Initiate colly
	collector := colly.NewCollector(
		//nolint:lll // allow long line for user agent
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	collector.Async = cfg.Async
	collector.MaxDepth = cfg.MaxDepth

	// Set HTTP request timeout
	collector.SetRequestTimeout(time.Duration(cfg.Timeout) * time.Second)

	// Configure robots.txt behavior
	if !cfg.RespectRobotsTxt {
		collector.IgnoreRobotsTxt = true
	}

	// Configure rate limiting
	if cfg.RateLimitDelay > 0 || cfg.Parallelism > 0 {
		parallelism := cfg.Parallelism
		if parallelism <= 0 {
			parallelism = defaultParallelism
		}

		_ = collector.Limit(&colly.LimitRule{
			DomainGlob:  "*",
			Parallelism: parallelism,
			Delay:       cfg.RateLimitDelay,
			RandomDelay: cfg.RateLimitDelay / 2, //nolint:mnd // half of rate limit for randomization
		})
	}

	if cfg.Debug {
		collector.SetDebugger(&debug.LogDebugger{
			Output: os.Stderr,
			Prefix: "",
			Flag:   log.LstdFlags,
		})
	}

	scraper := &Scraper{
		cfg:       cfg,
		collector: collector,
		emailsSet: &emails{
			set: nil,
			m:   sync.Mutex{},
		},
	}

	// Configure retry with exponential backoff
	if cfg.MaxRetries > 0 {
		collector.OnError(func(r *colly.Response, err error) {
			retriesLeft, ok := r.Ctx.GetAny("retriesLeft").(int)
			if !ok {
				retriesLeft = cfg.MaxRetries
			}

			if retriesLeft > 0 {
				attempt := cfg.MaxRetries - retriesLeft + 1
				delay := cfg.RetryDelay * time.Duration(1<<uint(attempt-1)) //nolint:gosec // attempt is bounded by MaxRetries
				delay = min(delay, maxRetryDelay)

				scraper.log("retrying request to", r.Request.URL, "in", delay, "(", retriesLeft, "retries left)")
				time.Sleep(delay)

				r.Ctx.Put("retriesLeft", retriesLeft-1)
				_ = r.Request.Retry()
			} else {
				scraper.log("request to", r.Request.URL, "failed after", cfg.MaxRetries, "retries:", err)
			}
		})

		collector.OnRequest(func(r *colly.Request) {
			r.Ctx.Put("retriesLeft", cfg.MaxRetries)
		})
	}

	if cfg.EnableJavascript {
		scraper.collector.OnResponse(func(response *colly.Response) {
			err := initiateScrapingFromChrome(response, cfg.Timeout)
			if err != nil {
				scraper.log(err)

				return
			}
		})
	}

	// Parse emails on each downloaded page
	scraper.collector.OnScraped(func(response *colly.Response) {
		scraper.emailsSet.parseEmails(response.Body)
	})

	// Cloudflare encoded email support
	scraper.collector.OnHTML("span[data-cfemail]", func(el *colly.HTMLElement) {
		scraper.emailsSet.parseCloudflareEmail(el.Attr("data-cfemail"))
	})

	if cfg.Recursively {
		// Find and visit all links
		scraper.collector.OnHTML("a[href]", func(el *colly.HTMLElement) {
			scraper.log("visiting: ", el.Attr("href"))

			err := el.Request.Visit(el.Attr("href"))
			if err != nil {
				// Ignore already visited error, this appears too often
				var alreadyVisited *colly.AlreadyVisitedError
				if !errors.As(err, &alreadyVisited) {
					scraper.log("error while linking: ", err.Error())
				}
			}
		})
	}

	return scraper
}

func (s *Scraper) log(v ...any) {
	if s.cfg.Debug {
		log.Println(v...)
	}
}
