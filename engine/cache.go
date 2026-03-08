package engine

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	response  SearchResponse
	expiresAt time.Time
}

// Cache is a thread-safe in-memory cache with TTL for search results.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	done    chan struct{}
}

// NewCache creates a new cache with the given TTL.
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		done:    make(chan struct{}),
	}
	go c.cleanup()
	return c
}

// Close stops the background cleanup goroutine.
func (c *Cache) Close() {
	select {
	case <-c.done:
		// already closed
	default:
		close(c.done)
	}
}

func cacheKey(query string, page int, safeSearch bool) string {
	safe := "0"
	if safeSearch {
		safe = "1"
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", query, page, safe)))
	return fmt.Sprintf("%x", h[:16])
}

// Get retrieves a cached response. Returns the response and true if found and not expired.
func (c *Cache) Get(query string, page int, safeSearch bool) (SearchResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[cacheKey(query, page, safeSearch)]
	if !ok || time.Now().After(entry.expiresAt) {
		return SearchResponse{}, false
	}
	return entry.response, true
}

// Set stores a response in the cache.
func (c *Cache) Set(query string, page int, safeSearch bool, resp SearchResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[cacheKey(query, page, safeSearch)] = cacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.entries {
				if now.After(v.expiresAt) {
					delete(c.entries, k)
				}
			}
			c.mu.Unlock()
		}
	}
}
