package emailscraper

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/lawzava/go-tld"
)

const (
	// minEmailDomainParts is the minimum number of parts a valid email domain should have (domain.tld).
	minEmailDomainParts = 2
	// minTLDLength is the minimum length for a valid TLD.
	minTLDLength = 2
	// minCloudflareEmailLength is the minimum length for a valid Cloudflare-encoded email (2 hex chars for XOR key).
	minCloudflareEmailLength = 2
)

type emails struct {
	set map[string]struct{}
	m   sync.Mutex
}

func (s *emails) add(email string) {
	if !isValidEmail(email) {
		return
	}

	s.m.Lock()
	defer s.m.Unlock()

	if s.set == nil {
		s.set = make(map[string]struct{})
	}

	s.set[email] = struct{}{}
}

func (s *emails) toSlice() []string {
	s.m.Lock()
	defer s.m.Unlock()

	result := make([]string, 0, len(s.set))
	for email := range s.set {
		result = append(result, email)
	}

	return result
}

func (s *emails) reset() {
	s.m.Lock()
	defer s.m.Unlock()

	s.set = nil
}

// Initialize once.
var (
	reg = regexp.MustCompile(`([a-zA-Z0-9._-]+@([a-zA-Z0-9_-]+\.)+[a-zA-Z0-9_-]+)`)

	// Matches common obfuscation patterns: [AT], (at), {AT}, " AT ", etc.
	obfuscatedSeparators = regexp.MustCompile(`\s*[\[\(\{]?\s*[aA][tT]\s*[\]\)\}]?\s*`)
)

// Parse any *@*.* string and append to the slice.
func (s *emails) parseEmails(body []byte) {
	res := reg.FindAll(body, -1)

	for _, r := range res {
		s.add(string(r))
	}

	body = obfuscatedSeparators.ReplaceAll(body, []byte("@"))

	res = reg.FindAll(body, -1)
	for _, r := range res {
		s.add(string(r))
	}
}

func (s *emails) parseCloudflareEmail(cloudflareEncodedEmail string) {
	decodedEmail := decodeCloudflareEmail(cloudflareEncodedEmail)
	email := reg.FindString(decodedEmail)

	s.add(email)
}

func decodeCloudflareEmail(email string) string {
	// Need at least 2 characters for the XOR key
	if len(email) < minCloudflareEmailLength {
		return ""
	}

	var buffer bytes.Buffer

	xorKey, err := strconv.ParseInt(email[0:2], 16, 0)
	if err != nil {
		return ""
	}

	for n := 4; n < len(email)+2; n += 2 {
		charCode, err := strconv.ParseInt(email[n-2:n], 16, 0)
		if err != nil {
			continue
		}

		decodedChar := charCode ^ xorKey

		buffer.WriteRune(rune(decodedChar))
	}

	return buffer.String()
}

// Check if email looks valid.
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	split := strings.Split(email, ".")

	if len(split) < minEmailDomainParts {
		return false
	}

	ending := split[len(split)-1]

	if len(ending) < minTLDLength {
		return false
	}

	// check if TLD name actually exists and is not some image ending
	if !tld.IsValid(ending) {
		return false
	}

	_, err := strconv.Atoi(ending)

	return err != nil
}
