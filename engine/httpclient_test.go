package engine

import (
	"net/http"
	"testing"
)

func TestRandomUserAgent(t *testing.T) {
	ua := randomUserAgent()
	if ua == "" {
		t.Fatal("randomUserAgent returned empty string")
	}

	found := false
	for _, poolUA := range userAgentPool {
		if ua == poolUA {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("randomUserAgent returned %q, which is not in the pool", ua)
	}
}

func TestSetBrowserHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	setBrowserHeaders(req)

	checks := map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"DNT":                       "1",
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
	}

	for header, want := range checks {
		got := req.Header.Get(header)
		if got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	ua := req.Header.Get("User-Agent")
	if ua == "" {
		t.Error("User-Agent header is empty")
	}
}
