package engine

import (
	"net/http"
	"testing"
	"time"
)

// mockProvider is a test double that returns predetermined results.
type mockProvider struct {
	name    string
	results []Result
	err     error
	delay   time.Duration
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Search(query string, page int) ([]Result, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.results, m.err
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"strips fragment", "https://example.com/page#section", "https://example.com/page"},
		{"strips trailing slash", "https://example.com/page/", "https://example.com/page"},
		{"lowercases host", "https://EXAMPLE.COM/Path", "https://example.com/Path"},
		{"preserves query params", "https://example.com/search?q=test", "https://example.com/search?q=test"},
		{"handles empty string", "", ""},
		{"handles malformed URL gracefully", "not-a-url", "not-a-url"},
		{"strips fragment and trailing slash", "https://example.com/page/#top", "https://example.com/page"},
		{"preserves path case", "https://example.com/CaseSensitive", "https://example.com/CaseSensitive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMergeAndRank_DeduplicatesByURL(t *testing.T) {
	eng := &Engine{MaxResults: 20}
	results := eng.mergeAndRank([]providerResult{
		{provider: "ddg", results: []Result{
			{URL: "https://example.com", Title: "Example", Snippet: "Short", Source: "DuckDuckGo"},
		}},
		{provider: "bing", results: []Result{
			{URL: "https://example.com", Title: "Example", Snippet: "A longer snippet here", Source: "Bing"},
		}},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 merged result, got %d", len(results))
	}
	if results[0].Snippet != "A longer snippet here" {
		t.Errorf("expected longer snippet to be kept, got %q", results[0].Snippet)
	}
}

func TestMergeAndRank_BoostsMultiSourceResults(t *testing.T) {
	eng := &Engine{MaxResults: 20}

	singleSource := []providerResult{
		{provider: "ddg", results: []Result{
			{URL: "https://unique.com", Title: "Unique", Source: "DuckDuckGo"},
		}},
	}

	multiSource := []providerResult{
		{provider: "ddg", results: []Result{
			{URL: "https://shared.com", Title: "Shared", Source: "DuckDuckGo"},
		}},
		{provider: "bing", results: []Result{
			{URL: "https://shared.com", Title: "Shared", Source: "Bing"},
		}},
	}

	singleResults := eng.mergeAndRank(singleSource)
	multiResults := eng.mergeAndRank(multiSource)

	if len(singleResults) == 0 || len(multiResults) == 0 {
		t.Fatal("expected results from both merges")
	}

	if multiResults[0].Score <= singleResults[0].Score {
		t.Errorf("multi-source score (%f) should be higher than single-source (%f)",
			multiResults[0].Score, singleResults[0].Score)
	}
}

func TestMergeAndRank_RespectsMaxResults(t *testing.T) {
	eng := &Engine{MaxResults: 3}

	var results []Result
	for i := 0; i < 10; i++ {
		results = append(results, Result{
			URL:    "https://example.com/" + string(rune('a'+i)),
			Title:  "Result",
			Source: "Test",
		})
	}

	resp := eng.mergeAndRank([]providerResult{
		{provider: "test", results: results},
	})

	// mergeAndRank itself doesn't limit — Search does. But let's verify it returns all.
	if len(resp) != 10 {
		t.Errorf("mergeAndRank returned %d results, expected 10 (Search limits, not mergeAndRank)", len(resp))
	}
}

func TestSearch_ReturnsResponse(t *testing.T) {
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name: "MockA",
				results: []Result{
					{URL: "https://a.com", Title: "Result A", Snippet: "Snippet A", Source: "MockA"},
				},
			},
			&mockProvider{
				name: "MockB",
				results: []Result{
					{URL: "https://b.com", Title: "Result B", Snippet: "Snippet B", Source: "MockB"},
				},
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
	}

	resp := eng.Search("test query", 1)

	if resp.Query != "test query" {
		t.Errorf("query = %q, want 'test query'", resp.Query)
	}
	if resp.Page != 1 {
		t.Errorf("page = %d, want 1", resp.Page)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
	if resp.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestSearch_LimitsResults(t *testing.T) {
	var results []Result
	for i := 0; i < 15; i++ {
		results = append(results, Result{
			URL:    "https://example.com/" + string(rune('a'+i)),
			Title:  "Result",
			Source: "Mock",
		})
	}

	eng := &Engine{
		Providers:  []Provider{&mockProvider{name: "Mock", results: results}},
		Timeout:    5 * time.Second,
		MaxResults: 5,
	}

	resp := eng.Search("test", 1)
	if len(resp.Results) != 5 {
		t.Errorf("expected 5 results (MaxResults), got %d", len(resp.Results))
	}
}

func TestSearch_HandlesProviderError(t *testing.T) {
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name: "Good",
				results: []Result{
					{URL: "https://good.com", Title: "Good Result", Source: "Good"},
				},
			},
			&mockProvider{
				name: "Bad",
				err:  http.ErrServerClosed,
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
	}

	resp := eng.Search("test", 1)

	if len(resp.Results) == 0 {
		t.Error("expected results from the working provider even when one fails")
	}
}

func TestSearch_HandlesTimeout(t *testing.T) {
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name: "Fast",
				results: []Result{
					{URL: "https://fast.com", Title: "Fast", Source: "Fast"},
				},
			},
			&mockProvider{
				name:  "Slow",
				delay: 3 * time.Second,
				results: []Result{
					{URL: "https://slow.com", Title: "Slow", Source: "Slow"},
				},
			},
		},
		Timeout:    500 * time.Millisecond,
		MaxResults: 20,
	}

	start := time.Now()
	resp := eng.Search("test", 1)
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("search took %v, expected timeout around 500ms", elapsed)
	}

	// Should have at least the fast result
	if len(resp.Results) == 0 {
		t.Error("expected at least fast provider results before timeout")
	}
}

func TestSearch_EmptyProviders(t *testing.T) {
	eng := &Engine{
		Providers:  []Provider{},
		Timeout:    5 * time.Second,
		MaxResults: 20,
	}

	resp := eng.Search("test", 1)
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results with no providers, got %d", len(resp.Results))
	}
}

func TestSearch_AllProvidersReturnEmpty(t *testing.T) {
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{name: "Empty1", results: nil},
			&mockProvider{name: "Empty2", results: []Result{}},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
	}

	resp := eng.Search("obscure query", 1)
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestMergeAndRank_PositionScoring(t *testing.T) {
	eng := &Engine{MaxResults: 20}
	results := eng.mergeAndRank([]providerResult{
		{provider: "test", results: []Result{
			{URL: "https://first.com", Title: "First", Source: "Test"},
			{URL: "https://second.com", Title: "Second", Source: "Test"},
			{URL: "https://third.com", Title: "Third", Source: "Test"},
		}},
	})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// First result (position 0) should have higher score than last (position 2)
	if results[0].Score <= results[2].Score {
		t.Errorf("first position score (%f) should be > last position score (%f)",
			results[0].Score, results[2].Score)
	}
}
