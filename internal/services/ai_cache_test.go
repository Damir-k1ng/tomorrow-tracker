package services

import (
	"sync"
	"testing"
	"time"
)

func TestAnswerCache_GetSetAndExpiry(t *testing.T) {
	t.Parallel()
	c := newAnswerCache(50*time.Millisecond, 10)
	c.Set("k1", "v1")

	if got, ok := c.Get("k1"); !ok || got != "v1" {
		t.Fatalf("immediate Get: ok=%v got=%q", ok, got)
	}

	time.Sleep(75 * time.Millisecond)
	if _, ok := c.Get("k1"); ok {
		t.Fatalf("expected entry to expire after TTL")
	}
}

func TestAnswerCache_NilSafe(t *testing.T) {
	t.Parallel()
	var c *answerCache
	if v, ok := c.Get("anything"); ok || v != "" {
		t.Fatalf("nil cache Get should return ('', false), got (%q, %v)", v, ok)
	}
	// Set on nil cache must not panic.
	c.Set("k", "v")
}

func TestAnswerCache_Eviction(t *testing.T) {
	t.Parallel()
	c := newAnswerCache(1*time.Hour, 4)
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3")
	c.Set("d", "4")
	if c.Stats() != 4 {
		t.Fatalf("expected 4 entries, got %d", c.Stats())
	}
	c.Set("e", "5") // triggers eviction
	if c.Stats() > 4 {
		t.Fatalf("after adding 5th entry to cap-4 cache, expected <=4, got %d", c.Stats())
	}
	// newest entry must survive
	if v, ok := c.Get("e"); !ok || v != "5" {
		t.Fatalf("newest entry evicted: got=(%q, %v)", v, ok)
	}
}

func TestCacheKey_Stable(t *testing.T) {
	t.Parallel()
	k1 := cacheKey(ModeSolve, "помоги решить pointone")
	k2 := cacheKey(ModeSolve, "помоги решить pointone")
	if k1 != k2 {
		t.Fatalf("same input should hash to same key")
	}
	k3 := cacheKey(ModeExplain, "помоги решить pointone")
	if k1 == k3 {
		t.Fatalf("different mode should produce different key")
	}
	k4 := cacheKey(ModeSolve, "помоги решить swap")
	if k1 == k4 {
		t.Fatalf("different question should produce different key")
	}
}

func TestAnswerCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	c := newAnswerCache(1*time.Hour, 100)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.Set("k", "v")
		}(i)
		go func() {
			defer wg.Done()
			_, _ = c.Get("k")
		}()
	}
	wg.Wait()
	// race detector catches the rest; just sanity-check state is consistent.
	if v, ok := c.Get("k"); !ok || v != "v" {
		t.Fatalf("post-concurrent state: got=(%q, %v)", v, ok)
	}
}
