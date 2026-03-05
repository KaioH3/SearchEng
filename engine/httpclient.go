package engine

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

var userAgentPool = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
}

func randomUserAgent() string {
	return userAgentPool[rand.Intn(len(userAgentPool))]
}

// NewRetryTransport creates a transport that retries on 429, 503, 5xx, and network errors.
func NewRetryTransport(base http.RoundTripper, maxRetries int, baseDelay time.Duration) http.RoundTripper {
	return &retryTransport{base: base, maxRetries: maxRetries, baseDelay: baseDelay}
}

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := t.baseDelay * (1 << (attempt - 1))
			jitter := time.Duration(rand.Int63n(int64(delay / 2)))
			select {
			case <-time.After(delay + jitter):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}

		resp, err = t.base.RoundTrip(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode == 503 || resp.StatusCode >= 500 {
			if attempt < t.maxRetries {
				resp.Body.Close()
				continue
			}
			return resp, nil
		}

		return resp, nil
	}

	return resp, fmt.Errorf("request failed after %d retries: %w", t.maxRetries+1, err)
}

// NewRateLimitedTransport creates a transport that rate-limits requests.
func NewRateLimitedTransport(base http.RoundTripper, limiter *rate.Limiter) http.RoundTripper {
	return &rateLimitedTransport{base: base, limiter: limiter}
}

type rateLimitedTransport struct {
	base    http.RoundTripper
	limiter *rate.Limiter
}

func (t *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req)
}

func setBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}
