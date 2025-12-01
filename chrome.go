package emailscraper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly/v2"
)

const (
	// defaultChromeWindowWidth is the default width for Chrome browser window.
	defaultChromeWindowWidth = 1920
	// defaultChromeWindowHeight is the default height for Chrome browser window.
	defaultChromeWindowHeight = 1080
)

// findChromePath attempts to find Chrome/Chromium executable.
func findChromePath() string {
	// Check environment variable first
	if path := os.Getenv("CHROME_PATH"); path != "" {
		return path
	}

	// Try to find chromium or chrome in PATH
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "chrome"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path
		}
	}

	return ""
}

func initiateScrapingFromChrome(response *colly.Response, timeout int) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(defaultChromeWindowWidth, defaultChromeWindowHeight),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	// Add custom Chrome path if found
	if chromePath := findChromePath(); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	defer func() {
		// Graceful shutdown waits for browser to close
		_ = chromedp.Cancel(allocCtx)

		allocCancel()
	}()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	if timeout > 0 {
		var timeoutCancel context.CancelFunc

		ctx, timeoutCancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)

		defer timeoutCancel()
	}

	var res string

	err := chromedp.Run(ctx,
		chromedp.Navigate(response.Request.URL.String()),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.InnerHTML("html", &res),
	)
	if err != nil {
		return fmt.Errorf("chromedp execution: %w", err)
	}

	response.Body = []byte(res)

	return nil
}
