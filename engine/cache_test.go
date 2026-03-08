package engine

import (
	"sync"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache(1 * time.Minute)
	defer c.Close()
	resp := SearchResponse{Query: "test", Page: 1}
	c.Set("test", 1, true, resp)

	got, ok := c.Get("test", 1, true)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Query != "test" {
		t.Errorf("query = %q, want 'test'", got.Query)
	}
}

func TestCache_Miss(t *testing.T) {
	c := NewCache(1 * time.Minute)
	defer c.Close()
	_, ok := c.Get("nonexistent", 1, true)
	if ok {
		t.Error("expected cache miss")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	c := NewCache(50 * time.Millisecond)
	defer c.Close()
	c.Set("test", 1, true, SearchResponse{Query: "test"})

	// Should hit immediately
	_, ok := c.Get("test", 1, true)
	if !ok {
		t.Fatal("expected cache hit before expiration")
	}

	time.Sleep(60 * time.Millisecond)

	// Should miss after TTL
	_, ok = c.Get("test", 1, true)
	if ok {
		t.Error("expected cache miss after TTL expiration")
	}
}

func TestCache_DifferentPages(t *testing.T) {
	c := NewCache(1 * time.Minute)
	defer c.Close()
	c.Set("test", 1, true, SearchResponse{Query: "test", Page: 1})
	c.Set("test", 2, true, SearchResponse{Query: "test", Page: 2})

	got1, ok1 := c.Get("test", 1, true)
	got2, ok2 := c.Get("test", 2, true)

	if !ok1 || !ok2 {
		t.Fatal("expected both pages to be cached")
	}
	if got1.Page != 1 || got2.Page != 2 {
		t.Errorf("pages = %d, %d; want 1, 2", got1.Page, got2.Page)
	}
}

func TestCache_DifferentSafeSearch(t *testing.T) {
	c := NewCache(1 * time.Minute)
	defer c.Close()
	c.Set("test", 1, true, SearchResponse{Query: "test", Answer: "safe"})
	c.Set("test", 1, false, SearchResponse{Query: "test", Answer: "unsafe"})

	gotSafe, okSafe := c.Get("test", 1, true)
	gotUnsafe, okUnsafe := c.Get("test", 1, false)

	if !okSafe || !okUnsafe {
		t.Fatal("expected both safe/unsafe to be cached separately")
	}
	if gotSafe.Answer != "safe" || gotUnsafe.Answer != "unsafe" {
		t.Errorf("answers = %q, %q; want 'safe', 'unsafe'", gotSafe.Answer, gotUnsafe.Answer)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := NewCache(1 * time.Minute)
	defer c.Close()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.Set("query", i%5, true, SearchResponse{Query: "query", Page: i % 5})
		}(i)
		go func(i int) {
			defer wg.Done()
			c.Get("query", i%5, true)
		}(i)
	}

	wg.Wait()
}
