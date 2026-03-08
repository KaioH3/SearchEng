package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSearch_ConcurrentSafe(t *testing.T) {
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name: "Mock",
				results: []Result{
					{URL: "https://example.com", Title: "Test result", Snippet: "Test concurrent content", Source: "Mock"},
				},
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
		SafeSearch: true,
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			safe := i%2 == 0
			opts := SearchOptions{SafeSearch: &safe}
			resp := eng.Search(context.Background(), "test query", 1, opts)
			if resp.Query != "test query" {
				t.Errorf("goroutine %d: query = %q", i, resp.Query)
			}
		}(i)
	}
	wg.Wait()
}

func TestSearch_ConcurrentWithCache(t *testing.T) {
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name:  "Mock",
				delay: 10 * time.Millisecond,
				results: []Result{
					{URL: "https://example.com", Title: "Test cached result", Snippet: "Test cache concurrent content", Source: "Mock"},
				},
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
		SafeSearch: true,
		Cache:      NewCache(1 * time.Minute),
	}
	defer eng.Cache.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := eng.Search(context.Background(), "cached test", 1)
			if resp.Query != "cached test" {
				t.Errorf("query = %q", resp.Query)
			}
		}()
	}
	wg.Wait()
}

func TestSearch_ConcurrentDifferentSafeSearch(t *testing.T) {
	cache := NewCache(1 * time.Minute)
	defer cache.Close()
	eng := &Engine{
		Providers: []Provider{
			&mockProvider{
				name: "Mock",
				results: []Result{
					{URL: "https://example.com", Title: "Test result", Snippet: "Test content about Go programming language", Source: "Mock"},
				},
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
		SafeSearch: true,
		Cache:      cache,
	}

	// Run concurrent searches with different SafeSearch values
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			safe := i%2 == 0
			opts := SearchOptions{SafeSearch: &safe}
			eng.Search(context.Background(), "test", 1, opts)
		}(i)
	}
	wg.Wait()
}

func TestGoogle_CooldownConcurrent(t *testing.T) {
	g := &Google{}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.startCooldown()
		}()
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.mu.Lock()
			_ = g.cooldownUntil
			g.mu.Unlock()
		}()
	}
	wg.Wait()
}

func TestMergeAndRank_ConcurrentSafe(t *testing.T) {
	eng := &Engine{MaxResults: 20}

	provResults := []providerResult{
		{provider: "ddg", results: []Result{
			{URL: "https://a.com", Title: "Test A result", Snippet: "Content about test A topic", Source: "DuckDuckGo"},
			{URL: "https://b.com", Title: "Test B result", Snippet: "Content about test B topic", Source: "DuckDuckGo"},
		}},
		{provider: "bing", results: []Result{
			{URL: "https://a.com", Title: "Test A result", Snippet: "Content about test A topic from Bing", Source: "Bing"},
		}},
	}

	// mergeAndRank itself isn't concurrent but verify it doesn't panic under repeated calls
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := eng.mergeAndRank(provResults, "test")
			if len(results) == 0 {
				t.Error("expected results")
			}
		}()
	}
	wg.Wait()
}
