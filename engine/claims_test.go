package engine

import (
	"math"
	"testing"
)

func TestExtractClaims_WithNumbers(t *testing.T) {
	results := []Result{
		{
			Source:  "DuckDuckGo",
			Snippet: "Go was released in 2009. It has over 2 million developers worldwide.",
		},
		{
			Source:  "Bing",
			Snippet: "Go was released in 2009 by Google. The language is used by many companies.",
		},
	}

	claims := ExtractClaims(results, false)
	if len(claims) == 0 {
		t.Fatal("expected claims to be extracted")
	}

	// The "released in 2009" claim should be corroborated
	found := false
	for _, c := range claims {
		if c.Corroboration > 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one corroborated claim")
	}
}

func TestExtractClaims_Definition(t *testing.T) {
	results := []Result{
		{
			Source:  "Mock",
			Snippet: "Go is a statically typed compiled language designed at Google.",
		},
	}

	claims := ExtractClaims(results, false)
	if len(claims) == 0 {
		t.Fatal("expected definition claim")
	}
}

func TestExtractClaims_NoClaimsInShortSnippet(t *testing.T) {
	results := []Result{
		{Source: "Mock", Snippet: "Go rocks."},
	}

	claims := ExtractClaims(results, false)
	if len(claims) != 0 {
		t.Errorf("expected no claims from short snippet, got %d", len(claims))
	}
}

func TestExtractClaims_EmptyResults(t *testing.T) {
	claims := ExtractClaims(nil, false)
	if claims != nil {
		t.Errorf("expected nil for empty results, got %v", claims)
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want float64
	}{
		{"identical", []string{"go", "programming"}, []string{"go", "programming"}, 1.0},
		{"disjoint", []string{"go"}, []string{"python"}, 0.0},
		{"partial", []string{"go", "programming", "language"}, []string{"go", "programming"}, 2.0 / 3.0},
		{"empty a", nil, []string{"go"}, 0.0},
		{"empty b", []string{"go"}, nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jaccardSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("jaccardSimilarity = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestClaimStrength(t *testing.T) {
	tests := []struct {
		sentence string
		wantZero bool
	}{
		{"Go was released in 2009 by Google", false},         // date + number
		{"It is 50% faster than Python", false},              // number + comparison
		{"Go is a programming language", false},              // definition
		{"According to the survey results show growth", false}, // attribution
		{"Hello world", true},                                // too short
		{"Nothing special about this sentence at all", true}, // no patterns
	}

	for _, tt := range tests {
		got := claimStrength(tt.sentence)
		if tt.wantZero && got != 0 {
			t.Errorf("claimStrength(%q) = %f, want 0", tt.sentence, got)
		}
		if !tt.wantZero && got == 0 {
			t.Errorf("claimStrength(%q) = 0, want > 0", tt.sentence)
		}
	}
}

func TestExtractClaims_ConfidenceBounded(t *testing.T) {
	results := []Result{
		{Source: "A", URL: "https://github.com/test", Snippet: "Go was released in 2009 with version 1.0 officially announced."},
		{Source: "B", URL: "https://wikipedia.org/go", Snippet: "Go was released in 2009 and version 1.0 was officially announced."},
		{Source: "C", URL: "https://go.dev/doc", Snippet: "Go was released in 2009 at Google and the first version was announced."},
		{Source: "D", URL: "https://example.com", Snippet: "Go was released in 2009 by the Go team at Google officially."},
	}

	claims := ExtractClaims(results, false)
	for _, c := range claims {
		if c.Confidence > 1.0 {
			t.Errorf("confidence %f exceeds 1.0 for claim: %s", c.Confidence, c.Text)
		}
		if c.Confidence < 0 {
			t.Errorf("confidence %f is negative for claim: %s", c.Confidence, c.Text)
		}
	}
}

func TestExtractClaims_TrustedSourceBoost(t *testing.T) {
	untrusted := []Result{
		{Source: "Mock", URL: "https://random.xyz/page", Snippet: "Go is a programming language created by Google in 2009."},
	}
	trusted := []Result{
		{Source: "Mock", URL: "https://github.com/golang/go", Snippet: "Go is a programming language created by Google in 2009."},
	}

	claimsUntrusted := ExtractClaims(untrusted, false)
	claimsTrusted := ExtractClaims(trusted, false)

	if len(claimsUntrusted) == 0 || len(claimsTrusted) == 0 {
		t.Fatal("expected claims from both")
	}

	if claimsTrusted[0].Confidence <= claimsUntrusted[0].Confidence {
		t.Errorf("trusted confidence (%f) should be > untrusted (%f)",
			claimsTrusted[0].Confidence, claimsUntrusted[0].Confidence)
	}
}

func TestExtractClaims_SafeSearchFiltersNSFW(t *testing.T) {
	results := []Result{
		{Source: "NSFW", URL: "https://xvideos.com/page", Snippet: "Go is a programming language released in 2009."},
		{Source: "Clean", URL: "https://golang.org/doc", Snippet: "Go is an open source language designed at Google in 2009."},
	}

	claimsSafe := ExtractClaims(results, true)
	claimsUnsafe := ExtractClaims(results, false)

	// Safe should have fewer sources since NSFW is filtered
	for _, c := range claimsSafe {
		for _, s := range c.Sources {
			if s == "NSFW" {
				t.Error("safe search should filter NSFW sources from claims")
			}
		}
	}

	// Unsafe should include NSFW sources
	if len(claimsUnsafe) == 0 {
		t.Fatal("expected claims without safe search")
	}
}

func TestExtractClaims_SafeSearchFiltersExplicitContent(t *testing.T) {
	results := []Result{
		{Source: "Mock", URL: "https://example.com/page", Snippet: "This porn site was launched in 2005 with over 1 million users."},
		{Source: "Mock2", URL: "https://example2.com/page", Snippet: "Go is a programming language released in 2009."},
	}

	claims := ExtractClaims(results, true)
	for _, c := range claims {
		if containsExplicit(c.Text) {
			t.Errorf("safe search should filter explicit content from claims: %q", c.Text)
		}
	}
}
