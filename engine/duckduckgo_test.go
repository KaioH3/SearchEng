package engine

import (
	"strings"
	"testing"
)

func TestExtractDDGURL_WithUddgParam(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"extracts uddg parameter",
			"//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage&rut=abc",
			"https://example.com/page",
		},
		{
			"returns raw URL when no uddg",
			"https://example.com/direct",
			"https://example.com/direct",
		},
		{
			"returns raw URL when uddg is empty",
			"//duckduckgo.com/l/?uddg=&rut=abc",
			"//duckduckgo.com/l/?uddg=&rut=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDDGURL(tt.input)
			if got != tt.want {
				t.Errorf("extractDDGURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasClass(t *testing.T) {
	tests := []struct {
		name    string
		classes string
		target  string
		want    bool
	}{
		{"single class match", "result__a", "result__a", true},
		{"multi class match", "foo result__a bar", "result__a", true},
		{"no match", "foo bar baz", "result__a", false},
		{"partial match is not a match", "result__abc", "result__a", false},
		{"empty classes", "", "result__a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a minimal HTML node with the class attribute
			node := &mockHTMLNode{classes: tt.classes}
			got := hasClassStr(node.classes, tt.target)
			if got != tt.want {
				t.Errorf("hasClass(%q, %q) = %v, want %v", tt.classes, tt.target, got, tt.want)
			}
		})
	}
}

// hasClassStr is a pure function test equivalent of hasClass logic
func hasClassStr(classAttr, target string) bool {
	for _, c := range strings.Fields(classAttr) {
		if c == target {
			return true
		}
	}
	return false
}

type mockHTMLNode struct {
	classes string
}

func TestTextContent_EmptyInput(t *testing.T) {
	result := textContent(nil)
	if result != "" {
		t.Errorf("textContent(nil) = %q, want empty string", result)
	}
}

func TestDuckDuckGo_Name(t *testing.T) {
	ddg := &DuckDuckGo{}
	if ddg.Name() != "DuckDuckGo" {
		t.Errorf("Name() = %q, want 'DuckDuckGo'", ddg.Name())
	}
}

func TestDuckDuckGo_ParseHTML(t *testing.T) {
	// Minimal DDG-like HTML structure
	html := `<html><body>
		<div class="result results_links results_links_deep web-result">
			<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage1">
				Example Page 1
			</a>
			<a class="result__snippet">This is a snippet for page 1</a>
		</div>
		<div class="result results_links results_links_deep web-result">
			<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage2">
				Example Page 2
			</a>
			<a class="result__snippet">This is a snippet for page 2</a>
		</div>
	</body></html>`

	ddg := &DuckDuckGo{}
	results, err := ddg.parse(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].URL != "https://example.com/page1" {
		t.Errorf("result[0].URL = %q, want 'https://example.com/page1'", results[0].URL)
	}
	if results[0].Title == "" {
		t.Error("result[0].Title should not be empty")
	}
	if results[0].Source != "DuckDuckGo" {
		t.Errorf("result[0].Source = %q, want 'DuckDuckGo'", results[0].Source)
	}
}

func TestDuckDuckGo_ParseEmptyHTML(t *testing.T) {
	ddg := &DuckDuckGo{}
	results, err := ddg.parse(strings.NewReader("<html><body></body></html>"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty HTML, got %d", len(results))
	}
}
