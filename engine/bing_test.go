package engine

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBing_Name(t *testing.T) {
	b := &Bing{}
	if b.Name() != "Bing" {
		t.Errorf("Name() = %q, want 'Bing'", b.Name())
	}
}

func TestBing_ParseHTML(t *testing.T) {
	html := `<html><body>
		<ol id="b_results">
			<li class="b_algo">
				<h2><a href="https://example.com/bing1">Bing Result One</a></h2>
				<div class="b_caption"><p>Snippet for bing result one</p></div>
			</li>
			<li class="b_algo">
				<h2><a href="https://example.com/bing2">Bing Result Two</a></h2>
				<div class="b_caption"><p>Snippet for bing result two</p></div>
			</li>
		</ol>
	</body></html>`

	b := &Bing{}
	results, err := b.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].URL != "https://example.com/bing1" {
		t.Errorf("result[0].URL = %q, want 'https://example.com/bing1'", results[0].URL)
	}
	if !strings.Contains(results[0].Title, "Bing Result One") {
		t.Errorf("result[0].Title = %q, expected to contain 'Bing Result One'", results[0].Title)
	}
	if results[0].Snippet != "Snippet for bing result one" {
		t.Errorf("result[0].Snippet = %q", results[0].Snippet)
	}
	if results[0].Source != "Bing" {
		t.Errorf("result[0].Source = %q, want 'Bing'", results[0].Source)
	}
}

func TestExtractBingURL(t *testing.T) {
	realURL := "https://example.com/page?q=test"
	encoded := "a1" + base64.RawURLEncoding.EncodeToString([]byte(realURL))
	trackingURL := "https://www.bing.com/ck/a?!&&p=abc&u=" + encoded + "&ntb=1"

	got := extractBingURL(trackingURL)
	if got != realURL {
		t.Errorf("extractBingURL() = %q, want %q", got, realURL)
	}
}

func TestExtractBingURL_Passthrough(t *testing.T) {
	directURL := "https://example.com/direct"
	got := extractBingURL(directURL)
	if got != directURL {
		t.Errorf("extractBingURL() = %q, want %q (passthrough)", got, directURL)
	}
}

func TestBing_ParseTrackingURLs(t *testing.T) {
	realURL := "https://example.com/real"
	encoded := "a1" + base64.RawURLEncoding.EncodeToString([]byte(realURL))
	html := `<html><body>
		<ol id="b_results">
			<li class="b_algo">
				<h2><a href="https://www.bing.com/ck/a?u=` + encoded + `&ntb=1">Tracked Result</a></h2>
				<div class="b_caption"><p>Snippet</p></div>
			</li>
		</ol>
	</body></html>`

	b := &Bing{}
	results, err := b.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].URL != realURL {
		t.Errorf("result URL = %q, want %q", results[0].URL, realURL)
	}
}

func TestBing_ParseEmptyHTML(t *testing.T) {
	b := &Bing{}
	results, err := b.parse(strings.NewReader("<html><body></body></html>"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty HTML, got %d", len(results))
	}
}

func TestExtractBingURL_MalformedURL(t *testing.T) {
	got := extractBingURL("://not-valid")
	if got != "://not-valid" {
		t.Errorf("expected passthrough for malformed URL, got %q", got)
	}
}

func TestExtractBingURL_NoUParam(t *testing.T) {
	got := extractBingURL("https://www.bing.com/ck/a?p=abc&ntb=1")
	if got != "https://www.bing.com/ck/a?p=abc&ntb=1" {
		t.Errorf("expected passthrough when no u param, got %q", got)
	}
}

func TestExtractBingURL_WithoutA1Prefix(t *testing.T) {
	realURL := "https://example.com/page"
	encoded := base64.RawURLEncoding.EncodeToString([]byte(realURL))
	trackingURL := "https://www.bing.com/ck/a?u=" + encoded + "&ntb=1"

	got := extractBingURL(trackingURL)
	if got != realURL {
		t.Errorf("extractBingURL() = %q, want %q", got, realURL)
	}
}

func TestExtractBingURL_InvalidBase64(t *testing.T) {
	trackingURL := "https://www.bing.com/ck/a?u=a1!!!invalid!!!&ntb=1"
	got := extractBingURL(trackingURL)
	if got != trackingURL {
		t.Errorf("expected passthrough for invalid base64, got %q", got)
	}
}
