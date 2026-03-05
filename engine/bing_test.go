package engine

import (
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
