package engine

import "time"

// Result represents a single search result from a provider.
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
	// Score is used internally for merge ranking. Higher is better.
	Score float64 `json:"-"`
}

// SearchResponse is the aggregated response returned to callers.
type SearchResponse struct {
	Query    string        `json:"query"`
	Results  []Result      `json:"results"`
	Duration time.Duration `json:"duration_ms"`
	Page     int           `json:"page"`
}
