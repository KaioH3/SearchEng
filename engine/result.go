package engine

import (
	"fmt"
	"net/url"
	"time"
)

// TrustSignals contains credibility indicators for a search result.
type TrustSignals struct {
	IsHTTPS       bool   `json:"is_https"`
	TLD           string `json:"tld"`
	TLDCategory   string `json:"tld_category"`
	IsTrusted     bool   `json:"is_trusted"`
	TrustedDomain string `json:"trusted_domain,omitempty"`
	SourceCount   int    `json:"source_count"`
}

// Claim represents a factual assertion extracted from search result snippets.
type Claim struct {
	Text          string   `json:"text"`
	Sources       []string `json:"sources"`
	Corroboration int      `json:"corroboration"`
	Confidence    float64  `json:"confidence"`
}

// Result represents a single search result from a provider.
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
	Favicon string `json:"favicon,omitempty"`
	// Score is used internally for merge ranking. Higher is better.
	Score float64       `json:"-"`
	Trust *TrustSignals `json:"-"`
}

// FaviconURL returns a Google favicon service URL for the result's domain.
func (r Result) FaviconURL() string {
	u, err := url.Parse(r.URL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=32", u.Host)
}

// ProviderStatus reports the outcome of a single provider query.
type ProviderStatus struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Error   string `json:"error,omitempty"`
}

// SearchResponse is the aggregated response returned to callers.
type SearchResponse struct {
	Query     string           `json:"query"`
	Results   []Result         `json:"results"`
	Answer    string           `json:"answer,omitempty"`
	Claims    []Claim          `json:"claims,omitempty"`
	Duration  time.Duration    `json:"duration_ms"`
	Page      int              `json:"page"`
	Providers []ProviderStatus `json:"providers,omitempty"`
}
