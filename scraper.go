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
	// randomDelayDivisor is used to calculate random delay as a fraction of rate limit delay.
	randomDelayDivisor = 2
	// defaultUserAgent is the default user agent string for HTTP requests.
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
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
	collector := colly.NewCollector(
		colly.UserAgent(defaultUserAgent),
	)

	configureCollector(collector, cfg)

	scraper := &Scraper{
		cfg:       cfg,
		collector: collector,
		emailsSet: &emails{
			set: nil,
			m:   sync.Mutex{},
		},
	}

	scraper.configureRetry()
	scraper.configureCallbacks()

	return scraper
}

// configureCollector sets up the collector with basic settings.
func configureCollector(collector *colly.Collector, cfg Config) {
	collector.Async = cfg.Async
	collector.MaxDepth = cfg.MaxDepth
	collector.SetRequestTimeout(time.Duration(cfg.Timeout) * time.Second)

	if !cfg.RespectRobotsTxt {
		collector.IgnoreRobotsTxt = true
	}

	if cfg.Debug {
		collector.SetDebugger(&debug.LogDebugger{
			Output: os.Stderr,
			Prefix: "",
			Flag:   log.LstdFlags,
		})
	}

	configureRateLimiting(collector, cfg)
}

// configureRateLimiting sets up rate limiting rules on the collector.
func configureRateLimiting(collector *colly.Collector, cfg Config) {
	if cfg.RateLimitDelay <= 0 && cfg.Parallelism <= 0 {
		return
	}

	parallelism := cfg.Parallelism
	if parallelism <= 0 {
		parallelism = defaultParallelism
	}

	_ = collector.Limit(&colly.LimitRule{
		DomainRegexp: "",
		DomainGlob:   "*",
		Delay:        cfg.RateLimitDelay,
		RandomDelay:  cfg.RateLimitDelay / randomDelayDivisor,
		Parallelism:  parallelism,
	})
}

// configureRetry sets up retry with exponential backoff.
func (s *Scraper) configureRetry() {
	if s.cfg.MaxRetries <= 0 {
		return
	}

	s.collector.OnError(func(response *colly.Response, err error) {
		s.handleRequestError(response, err)
	})

	s.collector.OnRequest(func(request *colly.Request) {
		request.Ctx.Put("retriesLeft", s.cfg.MaxRetries)
	})
}

// handleRequestError handles errors during requests and implements retry logic.
func (s *Scraper) handleRequestError(response *colly.Response, err error) {
	retriesLeft, ok := response.Ctx.GetAny("retriesLeft").(int)
	if !ok {
		retriesLeft = s.cfg.MaxRetries
	}

	if retriesLeft <= 0 {
		s.log("request to", response.Request.URL, "failed after", s.cfg.MaxRetries, "retries:", err)

		return
	}

	delay := s.calculateRetryDelay(retriesLeft)

	s.log("retrying request to", response.Request.URL, "in", delay, "(", retriesLeft, "retries left)")
	time.Sleep(delay)

	response.Ctx.Put("retriesLeft", retriesLeft-1)
	_ = response.Request.Retry()
}

// calculateRetryDelay computes the delay for the current retry attempt using exponential backoff.
func (s *Scraper) calculateRetryDelay(retriesLeft int) time.Duration {
	attempt := s.cfg.MaxRetries - retriesLeft + 1

	// Use bounded multiplication to avoid overflow
	// For typical MaxRetries (1-10), this is safe
	var multiplier int64 = 1

	for i := 1; i < attempt && i < 32; i++ {
		multiplier *= 2
		if multiplier > int64(maxRetryDelay/s.cfg.RetryDelay) {
			return maxRetryDelay
		}
	}

	delay := s.cfg.RetryDelay * time.Duration(multiplier)

	return min(delay, maxRetryDelay)
}

// configureCallbacks sets up all the collector callbacks for scraping.
func (s *Scraper) configureCallbacks() {
	if s.cfg.EnableJavascript {
		s.collector.OnResponse(func(response *colly.Response) {
			err := initiateScrapingFromChrome(response, s.cfg.Timeout)
			if err != nil {
				s.log(err)

				return
			}
		})
	}

	// Parse emails on each downloaded page
	s.collector.OnScraped(func(response *colly.Response) {
		s.emailsSet.parseEmails(response.Body)
	})

	// Cloudflare encoded email support
	s.collector.OnHTML("span[data-cfemail]", func(el *colly.HTMLElement) {
		s.emailsSet.parseCloudflareEmail(el.Attr("data-cfemail"))
	})

	if s.cfg.Recursively {
		s.configureRecursiveCrawling()
	}
}

// configureRecursiveCrawling sets up link following for recursive scraping.
func (s *Scraper) configureRecursiveCrawling() {
	s.collector.OnHTML("a[href]", func(el *colly.HTMLElement) {
		s.log("visiting: ", el.Attr("href"))

		err := el.Request.Visit(el.Attr("href"))
		if err != nil {
			// Ignore already visited error, this appears too often
			var alreadyVisited *colly.AlreadyVisitedError
			if !errors.As(err, &alreadyVisited) {
				s.log("error while linking: ", err.Error())
			}
		}
	})
}

func (s *Scraper) log(v ...any) {
	if s.cfg.Debug {
		log.Println(v...)
	}
}
