package engine

import (
	"context"
	"math"
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

func (m *mockProvider) Search(ctx context.Context, query string, page int) ([]Result, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
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
	}, "example")

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
			{URL: "https://unique.com", Title: "Unique shared page", Snippet: "Content about shared topic", Source: "DuckDuckGo"},
		}},
	}

	multiSource := []providerResult{
		{provider: "ddg", results: []Result{
			{URL: "https://shared.com", Title: "Shared page", Snippet: "Content about shared topic", Source: "DuckDuckGo"},
		}},
		{provider: "bing", results: []Result{
			{URL: "https://shared.com", Title: "Shared page", Snippet: "Content about shared topic", Source: "Bing"},
		}},
	}

	singleResults := eng.mergeAndRank(singleSource, "shared")
	multiResults := eng.mergeAndRank(multiSource, "shared")

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
			URL:     "https://example.com/" + string(rune('a'+i)),
			Title:   "Test Result",
			Snippet: "A test snippet with relevant content",
			Source:  "Test",
		})
	}

	resp := eng.mergeAndRank([]providerResult{
		{provider: "test", results: results},
	}, "test")

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
					{URL: "https://a.com", Title: "Test query result A", Snippet: "Content about test query topic A", Source: "MockA"},
				},
			},
			&mockProvider{
				name: "MockB",
				results: []Result{
					{URL: "https://b.com", Title: "Test query result B", Snippet: "Content about test query topic B", Source: "MockB"},
				},
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
	}

	resp := eng.Search(context.Background(), "test query", 1)

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
	if len(resp.Providers) != 2 {
		t.Errorf("expected 2 provider statuses, got %d", len(resp.Providers))
	}
}

func TestSearch_LimitsResults(t *testing.T) {
	var results []Result
	for i := 0; i < 15; i++ {
		results = append(results, Result{
			URL:     "https://example.com/" + string(rune('a'+i)),
			Title:   "Test Result",
			Snippet: "A test snippet with relevant content",
			Source:  "Mock",
		})
	}

	eng := &Engine{
		Providers:  []Provider{&mockProvider{name: "Mock", results: results}},
		Timeout:    5 * time.Second,
		MaxResults: 5,
	}

	resp := eng.Search(context.Background(), "test", 1)
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
					{URL: "https://good.com", Title: "Test good result", Snippet: "Test content for good result", Source: "Good"},
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

	resp := eng.Search(context.Background(), "test", 1)

	if len(resp.Results) == 0 {
		t.Error("expected results from the working provider even when one fails")
	}

	// Verify provider status reports the error
	var badStatus *ProviderStatus
	for _, ps := range resp.Providers {
		if ps.Name == "Bad" {
			ps := ps
			badStatus = &ps
		}
	}
	if badStatus == nil {
		t.Fatal("expected status for Bad provider")
	}
	if badStatus.Success {
		t.Error("expected Bad provider to report failure")
	}
}

func TestSearch_HandlesTimeout(t *testing.T) {
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name: "Fast",
				results: []Result{
					{URL: "https://fast.com", Title: "Fast test result", Snippet: "Test content for fast result", Source: "Fast"},
				},
			},
			&mockProvider{
				name:  "Slow",
				delay: 3 * time.Second,
				results: []Result{
					{URL: "https://slow.com", Title: "Slow test result", Snippet: "Test content for slow result", Source: "Slow"},
				},
			},
		},
		Timeout:    500 * time.Millisecond,
		MaxResults: 20,
	}

	start := time.Now()
	resp := eng.Search(context.Background(), "test", 1)
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

	resp := eng.Search(context.Background(), "test", 1)
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

	resp := eng.Search(context.Background(), "obscure query", 1)
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestMergeAndRank_PositionScoring(t *testing.T) {
	eng := &Engine{MaxResults: 20}
	results := eng.mergeAndRank([]providerResult{
		{provider: "test", results: []Result{
			{URL: "https://first.com", Title: "First test result", Snippet: "Content about test topic", Source: "Test"},
			{URL: "https://second.com", Title: "Second test result", Snippet: "Content about test topic", Source: "Test"},
			{URL: "https://third.com", Title: "Third test result", Snippet: "Content about test topic", Source: "Test"},
		}},
	}, "test")

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Score <= results[2].Score {
		t.Errorf("first position score (%f) should be > last position score (%f)",
			results[0].Score, results[2].Score)
	}
}

func TestSearch_CacheHit(t *testing.T) {
	callCount := 0
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name: "Mock",
				results: []Result{
					{URL: "https://cached.com", Title: "Test cache result", Snippet: "Test cache content", Source: "Mock"},
				},
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
		Cache:      NewCache(1 * time.Minute),
	}
	defer eng.Cache.Close()

	// Wrap provider to count calls
	orig := eng.Providers[0]
	eng.Providers[0] = &countingProvider{Provider: orig, count: &callCount}

	eng.Search(context.Background(), "test cache", 1)
	eng.Search(context.Background(), "test cache", 1)

	if callCount != 1 {
		t.Errorf("expected provider to be called once (cached), got %d", callCount)
	}
}

type countingProvider struct {
	Provider
	count *int
}

func (c *countingProvider) Search(ctx context.Context, query string, page int) ([]Result, error) {
	*c.count++
	return c.Provider.Search(ctx, query, page)
}

func (c *countingProvider) Name() string {
	return c.Provider.Name()
}

func TestBM25FScore(t *testing.T) {
	r := Result{Title: "Go programming language", Snippet: "Go is an open source programming language"}
	queryTerms := []string{"go", "programming"}
	df := map[string]int{"go": 3, "programming": 2}

	score := bm25fScore(r, queryTerms, df, 10)
	if score <= 0 {
		t.Errorf("expected positive BM25F score, got %f", score)
	}
}

func TestBM25FScore_EmptyQuery(t *testing.T) {
	r := Result{Title: "Test", Snippet: "content"}
	score := bm25fScore(r, nil, nil, 10)
	if score != 0 {
		t.Errorf("expected 0 for empty query, got %f", score)
	}
}

func TestBM25FScore_ZeroDocs(t *testing.T) {
	r := Result{Title: "Test", Snippet: "content"}
	score := bm25fScore(r, []string{"test"}, map[string]int{"test": 1}, 0)
	if score != 0 {
		t.Errorf("expected 0 for zero docs, got %f", score)
	}
}

func TestBM25FScore_TitleBoost(t *testing.T) {
	inTitle := Result{Title: "golang tutorial", Snippet: "learn programming"}
	notInTitle := Result{Title: "something else", Snippet: "golang tutorial here"}
	queryTerms := []string{"golang"}
	df := map[string]int{"golang": 2}

	scoreInTitle := bm25fScore(inTitle, queryTerms, df, 5)
	scoreNotInTitle := bm25fScore(notInTitle, queryTerms, df, 5)

	if scoreInTitle <= scoreNotInTitle {
		t.Errorf("title match score (%f) should be > non-title (%f)", scoreInTitle, scoreNotInTitle)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello world", 2},
		{"", 0},
		{"a b", 0},                // single-char words filtered
		{"Go is great!", 2},       // "is" is stopword → "go", "great"
		{`"quoted" (words)`, 2},
	}
	for _, tt := range tests {
		got := tokenize(tt.input)
		if len(got) != tt.want {
			t.Errorf("tokenize(%q) = %v (%d terms), want %d", tt.input, got, len(got), tt.want)
		}
	}
}

func TestTokenize_FiltersStopwords(t *testing.T) {
	got := tokenize("how to be the best")
	if len(got) != 1 || got[0] != "best" {
		t.Errorf("tokenize('how to be the best') = %v, want [best]", got)
	}

	got = tokenize("como ser engenheiro de software")
	// "como", "ser", "de" are stopwords
	if len(got) != 2 {
		t.Errorf("tokenize PT = %v, want [engenheiro software]", got)
	}
}

func TestRRF_MultiProviderBoost(t *testing.T) {
	eng := &Engine{MaxResults: 20}

	// Result in 3 providers should rank above result in 1
	results := eng.mergeAndRank([]providerResult{
		{provider: "ddg", results: []Result{
			{URL: "https://multi.com", Title: "Multi", Snippet: "Appears in multiple sources for testing", Source: "DuckDuckGo"},
			{URL: "https://single.com", Title: "Single", Snippet: "Appears in one source only for testing", Source: "DuckDuckGo"},
		}},
		{provider: "bing", results: []Result{
			{URL: "https://multi.com", Title: "Multi", Snippet: "Appears in multiple sources for testing", Source: "Bing"},
		}},
		{provider: "google", results: []Result{
			{URL: "https://multi.com", Title: "Multi", Snippet: "Appears in multiple sources for testing", Source: "Google"},
		}},
	}, "testing")

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	var multiScore, singleScore float64
	for _, r := range results {
		if r.URL == "https://multi.com" {
			multiScore = r.Score
		}
		if r.URL == "https://single.com" {
			singleScore = r.Score
		}
	}

	if multiScore <= singleScore {
		t.Errorf("multi-provider result score (%f) should be > single-provider (%f)", multiScore, singleScore)
	}
}

func TestExtractTLD(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/page", ".com"},
		{"https://mit.edu/research", ".edu"},
		{"https://example.gov/data", ".gov"},
		{"https://site.com.br/page", ".com.br"},
		{"https://example.co.uk/test", ".co.uk"},
		{"https://example.xyz", ".xyz"},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		got := extractTLD(tt.url)
		if got != tt.want {
			t.Errorf("extractTLD(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestTLDScore(t *testing.T) {
	weight := 1.0
	eduScore := tldScore("https://mit.edu/research", weight)
	comScore := tldScore("https://example.com/page", weight)
	xyzScore := tldScore("https://spam.xyz/page", weight)

	if eduScore <= comScore {
		t.Errorf(".edu score (%f) should be > .com score (%f)", eduScore, comScore)
	}
	if comScore <= xyzScore {
		t.Errorf(".com score (%f) should be > .xyz score (%f)", comScore, xyzScore)
	}
}

func TestHTTPSBonus(t *testing.T) {
	bonus := 0.3
	if got := httpsBonus("https://example.com", bonus); got != bonus {
		t.Errorf("httpsBonus(https) = %f, want %f", got, bonus)
	}
	if got := httpsBonus("http://example.com", bonus); got != 0 {
		t.Errorf("httpsBonus(http) = %f, want 0", got)
	}
}

func TestComputeTrustSignals(t *testing.T) {
	ts := computeTrustSignals("https://github.com/user/repo", []string{"DuckDuckGo", "Bing"})
	if !ts.IsHTTPS {
		t.Error("expected IsHTTPS=true")
	}
	if !ts.IsTrusted {
		t.Error("expected IsTrusted=true for github.com")
	}
	if ts.TrustedDomain != "github.com" {
		t.Errorf("TrustedDomain = %q, want 'github.com'", ts.TrustedDomain)
	}
	if ts.SourceCount != 2 {
		t.Errorf("SourceCount = %d, want 2", ts.SourceCount)
	}
	if ts.TLD != ".com" {
		t.Errorf("TLD = %q, want '.com'", ts.TLD)
	}
}

func TestTLDCategory(t *testing.T) {
	tests := []struct {
		tld  string
		want string
	}{
		{".edu", "academic"},
		{".gov", "government"},
		{".com", "commercial"},
		{".xyz", "spam-prone"},
		{".unknown", "other"},
	}
	for _, tt := range tests {
		got := tldCategory(tt.tld)
		if got != tt.want {
			t.Errorf("tldCategory(%q) = %q, want %q", tt.tld, got, tt.want)
		}
	}
}

func TestMergeAndRank_TrustSignalsPopulated(t *testing.T) {
	eng := &Engine{MaxResults: 20}
	results := eng.mergeAndRank([]providerResult{
		{provider: "test", results: []Result{
			{URL: "https://github.com/test", Title: "Test", Snippet: "Test snippet", Source: "Test"},
		}},
	}, "test")

	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].Trust == nil {
		t.Fatal("expected Trust signals to be populated")
	}
	if !results[0].Trust.IsHTTPS {
		t.Error("expected IsHTTPS=true")
	}
}

func TestExtractTLD_IPAddresses(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"IPv4", "http://192.168.1.1:8080/path", ""},
		{"IPv4 no port", "http://10.0.0.1/page", ""},
		{"IPv6", "http://[::1]:8080/path", ""},
		{"IPv6 full", "http://[2001:db8::1]/page", ""},
		{"localhost", "http://localhost/page", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTLD(tt.url)
			if got != tt.want {
				t.Errorf("extractTLD(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtractTLD_WithPort(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com:443/page", ".com"},
		{"https://site.co.uk:8080/test", ".co.uk"},
	}
	for _, tt := range tests {
		got := extractTLD(tt.url)
		if got != tt.want {
			t.Errorf("extractTLD(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestBM25FScore_URLOnlyMatches(t *testing.T) {
	r := Result{Title: "", Snippet: "", URL: "https://example.com/golang/tutorial"}
	queryTerms := []string{"golang"}
	df := map[string]int{"golang": 1}

	score := bm25fScore(r, queryTerms, df, 5)
	if score <= 0 {
		t.Errorf("expected positive BM25F score for URL match, got %f", score)
	}
}

func TestBM25FScore_AllFieldsMatch(t *testing.T) {
	r := Result{
		Title:   "golang tutorial",
		Snippet: "learn golang programming with examples",
		URL:     "https://example.com/golang/guide",
	}
	queryTerms := []string{"golang"}
	df := map[string]int{"golang": 2}

	score := bm25fScore(r, queryTerms, df, 5)
	// Should score higher than URL-only match
	urlOnly := Result{Title: "", Snippet: "", URL: "https://example.com/golang/guide"}
	urlScore := bm25fScore(urlOnly, queryTerms, df, 5)

	if score <= urlScore {
		t.Errorf("all-fields score (%f) should be > URL-only score (%f)", score, urlScore)
	}
}

func TestRRF_EmptyProviderResults(t *testing.T) {
	eng := &Engine{MaxResults: 20}
	results := eng.mergeAndRank([]providerResult{
		{provider: "empty", results: []Result{}},
	}, "test")

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty provider, got %d", len(results))
	}
}

func TestMergeAndRank_EDUvsXYZ(t *testing.T) {
	eng := &Engine{MaxResults: 20}
	results := eng.mergeAndRank([]providerResult{
		{provider: "test", results: []Result{
			{URL: "http://spam.xyz/page", Title: "Result", Snippet: "Some content about testing here", Source: "Test"},
			{URL: "https://mit.edu/research", Title: "Result", Snippet: "Some content about testing here", Source: "Test"},
		}},
	}, "testing")

	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].URL != "https://mit.edu/research" {
		t.Errorf("expected .edu HTTPS result first, got %s", results[0].URL)
	}
}

func TestTokenizeURL_Basic(t *testing.T) {
	tokens := tokenizeURL("https://example.com/golang/tutorial-guide")
	if len(tokens) == 0 {
		t.Fatal("expected tokens from URL path")
	}
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["golang"] {
		t.Error("expected 'golang' token")
	}
	if !found["tutorial"] {
		t.Error("expected 'tutorial' token")
	}
	if !found["guide"] {
		t.Error("expected 'guide' token")
	}
}

func TestTokenizeURL_InvalidURL(t *testing.T) {
	tokens := tokenizeURL("://not-valid")
	if tokens != nil {
		t.Errorf("expected nil for invalid URL, got %v", tokens)
	}
}

func TestTrustedDomainBonus(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://github.com/user/repo", true},
		{"https://en.wikipedia.org/wiki/Go", true},
		{"https://stackoverflow.com/questions/123", true},
		{"https://randomsite.com/page", false},
		{"not-a-url", false},
	}
	for _, tt := range tests {
		got := trustedDomainBonus(tt.url, 1.5)
		if (got > 0) != tt.want {
			t.Errorf("trustedDomainBonus(%q) = %f, wantBonus = %v", tt.url, got, tt.want)
		}
	}
}

func TestBM25FScore_NeverNaN(t *testing.T) {
	// Many docs where every term appears in every doc → IDF could go negative
	queryTerms := []string{"go", "programming", "language"}
	df := map[string]int{"go": 100, "programming": 100, "language": 100}
	totalDocs := 50.0 // docFreq > totalDocs → negative IDF without floor

	for i := 0; i < 100; i++ {
		r := Result{
			Title:   "Go programming language tutorial",
			Snippet: "Learn Go programming language basics",
			URL:     "https://example.com/go",
		}
		score := bm25fScore(r, queryTerms, df, totalDocs)
		if math.IsNaN(score) || math.IsInf(score, 0) {
			t.Fatalf("bm25fScore returned NaN/Inf: %f", score)
		}
		if score < 0 {
			t.Fatalf("bm25fScore returned negative: %f", score)
		}
	}
}

func TestMergeAndRank_NaNGuard(t *testing.T) {
	eng := &Engine{MaxResults: 20}
	// Edge case: single result, term appears in every doc
	results := eng.mergeAndRank([]providerResult{
		{provider: "test", results: []Result{
			{URL: "https://example.com", Title: "Test", Snippet: "Test content", Source: "Test"},
		}},
	}, "test")

	for _, r := range results {
		if math.IsNaN(r.Score) || math.IsInf(r.Score, 0) {
			t.Errorf("result score is NaN/Inf: %f", r.Score)
		}
	}
}

func TestQueryCoverage_LowCoverage(t *testing.T) {
	r := Result{Title: "Unrelated page", Snippet: "Nothing matching here"}
	queryTerms := []string{"golang", "tutorial", "advanced", "concurrency", "patterns", "best", "practices"}
	penalty := queryCoveragePenalty(r, queryTerms)
	if penalty != -2.0 {
		t.Errorf("queryCoveragePenalty = %f, want -2.0 for 0 matching terms", penalty)
	}
}

func TestQueryCoverage_GoodCoverage(t *testing.T) {
	r := Result{Title: "Golang tutorial", Snippet: "Advanced golang concurrency patterns"}
	queryTerms := []string{"golang", "tutorial", "advanced", "concurrency", "patterns"}
	penalty := queryCoveragePenalty(r, queryTerms)
	if penalty != 0 {
		t.Errorf("queryCoveragePenalty = %f, want 0 for good coverage", penalty)
	}
}

func TestLanguagePenalty_CJKSnippet(t *testing.T) {
	snippet := "这是一个测试页面关于搜索引擎优化"
	query := "como otimizar busca"
	penalty := languagePenalty(snippet, query)
	if penalty != -3.0 {
		t.Errorf("languagePenalty = %f, want -3.0 for CJK snippet with PT query", penalty)
	}
}

func TestLanguagePenalty_NoPenalty(t *testing.T) {
	snippet := "This is a normal English result about search engines"
	query := "search engines"
	penalty := languagePenalty(snippet, query)
	if penalty != 0 {
		t.Errorf("languagePenalty = %f, want 0 for matching language", penalty)
	}
}

func TestLanguagePenalty_CJKQuery(t *testing.T) {
	snippet := "这是一个测试"
	query := "这是搜索"
	penalty := languagePenalty(snippet, query)
	if penalty != 0 {
		t.Errorf("languagePenalty = %f, want 0 when query is also CJK", penalty)
	}
}

func TestCJKRatio(t *testing.T) {
	if r := cjkRatio("hello world"); r != 0 {
		t.Errorf("cjkRatio(english) = %f, want 0", r)
	}
	if r := cjkRatio("这是测试"); r < 0.9 {
		t.Errorf("cjkRatio(chinese) = %f, want >0.9", r)
	}
	if r := cjkRatio(""); r != 0 {
		t.Errorf("cjkRatio(empty) = %f, want 0", r)
	}
}
