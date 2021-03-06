![GolangCI](https://github.com/lawzava/emailscraper/workflows/golangci/badge.svg?branch=main)
[![Version](https://img.shields.io/badge/version-v1.0.0-green.svg)](https://github.com/lawzava/emailscraper/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/lawzava/emailscraper)](https://goreportcard.com/report/github.com/lawzava/emailscraper)
[![Coverage Status](https://coveralls.io/repos/github/lawzava/emailscraper/badge.svg?branch=main)](https://coveralls.io/github/lawzava/emailscraper?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/lawzava/emailscraper.svg)](https://pkg.go.dev/github.com/lawzava/emailscraper)

# emailscraper

Minimalistic library to scrape emails from websites.

Requires chromium or google-chrome available in environment for JS render utilization. 

## Installation

```
go get github.com/lawzava/emailscraper
```

## Usage

```go
package main

import (
	"fmt"
	
	"github.com/lawzava/emailscraper"
)

func main() {
	s := emailscraper.New(emailscraper.DefaultConfig())

	extractedEmails, err := s.Scrape("https://lawzava.com")
	if err != nil {
		panic(err)
	}
	
	fmt.Println(extractedEmails)
}
```
