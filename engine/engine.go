package engine

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// Engine is the meta-search aggregator that queries multiple providers in parallel.
type Engine struct {
	Providers  []Provider
	Timeout    time.Duration
	MaxResults int
}

// providerResult holds results from a single provider along with any error.
type providerResult struct {
	provider string
	results  []Result
	err      error
}

// Search queries all providers in parallel, deduplicates, and ranks results.
func (e *Engine) Search(query string, page int) SearchResponse {
	start := time.Now()
	timeout := e.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ch := make(chan providerResult, len(e.Providers))
	var wg sync.WaitGroup

	for _, p := range e.Providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			results, err := p.Search(query, page)
			ch <- providerResult{provider: p.Name(), results: results, err: err}
		}(p)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results with timeout
	var allResults []providerResult
	timer := time.NewTimer(timeout)
	defer timer.Stop()

collecting:
	for i := 0; i < len(e.Providers); i++ {
		select {
		case pr, ok := <-ch:
			if !ok {
				break collecting
			}
			if pr.err != nil {
				fmt.Printf("[warn] %s: %v\n", pr.provider, pr.err)
			}
			allResults = append(allResults, pr)
		case <-timer.C:
			fmt.Println("[warn] search timeout reached, returning partial results")
			break collecting
		}
	}

	merged := e.mergeAndRank(allResults)

	maxResults := e.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}
	if len(merged) > maxResults {
		merged = merged[:maxResults]
	}

	return SearchResponse{
		Query:    query,
		Results:  merged,
		Duration: time.Since(start),
		Page:     page,
	}
}

// mergeAndRank deduplicates results by URL and scores them.
// Results appearing in multiple providers get a higher score.
func (e *Engine) mergeAndRank(providerResults []providerResult) []Result {
	type scoredResult struct {
		result  Result
		sources []string
		score   float64
	}

	seen := make(map[string]*scoredResult)

	for _, pr := range providerResults {
		for i, r := range pr.results {
			normalized := normalizeURL(r.URL)
			if existing, ok := seen[normalized]; ok {
				// Boost score for appearing in multiple sources
				existing.sources = append(existing.sources, r.Source)
				existing.score += 10.0
				// Keep the longer snippet
				if len(r.Snippet) > len(existing.result.Snippet) {
					existing.result.Snippet = r.Snippet
				}
			} else {
				// Base score: position-based (earlier = higher)
				positionScore := float64(100-i) / 10.0
				seen[normalized] = &scoredResult{
					result:  r,
					sources: []string{r.Source},
					score:   positionScore,
				}
			}
		}
	}

	results := make([]Result, 0, len(seen))
	for _, sr := range seen {
		sr.result.Score = sr.score
		if len(sr.sources) > 1 {
			sr.result.Source = strings.Join(sr.sources, ", ")
		}
		results = append(results, sr.result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// normalizeURL strips trailing slashes, fragments, and lowercases the host for deduplication.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return strings.TrimRight(rawURL, "/")
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}
