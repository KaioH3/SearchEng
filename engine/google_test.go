package engine

import (
	"strings"
	"testing"
)

func TestGoogle_Name(t *testing.T) {
	g := &Google{}
	if g.Name() != "Google" {
		t.Errorf("Name() = %q, want 'Google'", g.Name())
	}
}

func TestGoogle_ParseHTML(t *testing.T) {
	html := `<html><body>
		<div class="g">
			<a href="https://example.com/result1"><h3>Result One</h3></a>
			<div class="VwiC3b">This is the snippet for result one</div>
		</div>
		<div class="g">
			<a href="https://example.com/result2"><h3>Result Two</h3></a>
			<div class="VwiC3b">Snippet for result two</div>
		</div>
	</body></html>`

	g := &Google{}
	results, err := g.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].URL != "https://example.com/result1" {
		t.Errorf("result[0].URL = %q, want 'https://example.com/result1'", results[0].URL)
	}
	if !strings.Contains(results[0].Title, "Result One") {
		t.Errorf("result[0].Title = %q, expected to contain 'Result One'", results[0].Title)
	}
	if results[0].Source != "Google" {
		t.Errorf("result[0].Source = %q, want 'Google'", results[0].Source)
	}
}

func TestGoogle_ParseEmptyHTML(t *testing.T) {
	g := &Google{}
	results, err := g.parse(strings.NewReader("<html><body></body></html>"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty HTML, got %d", len(results))
	}
}

func TestGoogle_ParseSkipsNoHref(t *testing.T) {
	html := `<html><body>
		<div class="g">
			<h3>Title Without Link</h3>
		</div>
	</body></html>`

	g := &Google{}
	results, err := g.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (no href), got %d", len(results))
	}
}
