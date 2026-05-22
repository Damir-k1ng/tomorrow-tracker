package services

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/knowledge"
)

// contentVersion is a short hex digest of every input that materially
// changes the model's output for a given (mode, question): the four
// system prompts plus the entire Piscine catalog (exercise solutions,
// samples, concepts, examples). Mixed into every cache key so that the
// moment we ship a prompt fix or RAG update, all existing cached
// answers become unreachable — no stale "wrong" replies linger for
// the cache TTL.
//
// Computed once at package init. Cheap (~100KB of strings through
// SHA-256). 12 hex chars (48 bits) is far beyond birthday-collision
// risk for our scale; we don't need full digest length here.
var contentVersion = computeContentVersion(
	[]string{promptSolve, promptExplain, promptReview, promptTutorial},
	knowledge.Exercises,
	knowledge.Concepts,
)

// computeContentVersion is the pure version of contentVersion exposed
// for testing — call sites in production should use the package-level
// `contentVersion`. Order of inputs is fixed and stable so the digest
// is reproducible across builds.
func computeContentVersion(prompts []string, exercises []knowledge.Exercise, concepts []knowledge.Concept) string {
	h := sha256.New()
	for _, p := range prompts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	for i := range exercises {
		ex := &exercises[i]
		h.Write([]byte(ex.Name))
		h.Write([]byte{0})
		h.Write([]byte(ex.Solution))
		h.Write([]byte{0})
		h.Write([]byte(ex.Samples))
		h.Write([]byte{0})
	}
	for i := range concepts {
		c := &concepts[i]
		h.Write([]byte(c.Name))
		h.Write([]byte{0})
		h.Write([]byte(c.Example))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// answerCache is the AI reply cache. Educational Q&A is highly repetitive
// — many students ask the same Piscine exercise the same way. Cached
// replies are returned in <1 ms instead of the 3-10 s round-trip to
// Pioneer, and obviously they don't cost any tokens.
//
// Tradeoffs picked here:
//   - Key is hash(mode + question) — history is intentionally ignored
//     so two different students asking "помоги решить pointone" share
//     the same cached answer. For our use case this is desirable; if
//     it ever becomes wrong, mix history into the hash.
//   - TTL is 1 hour. Long enough that a wave of identical questions
//     during a class hits cache, short enough that a model swap or
//     prompt edit clears stale answers naturally.
//   - LRU on size — bounded memory, oldest entries drop when we hit
//     maxEntries. The eviction is a periodic full sweep rather than
//     per-write to keep the hot path branch-free.
type answerCache struct {
	mu         sync.RWMutex
	entries    map[string]cacheEntry
	ttl        time.Duration
	maxEntries int
}

type cacheEntry struct {
	answer    string
	createdAt time.Time
}

func newAnswerCache(ttl time.Duration, maxEntries int) *answerCache {
	c := &answerCache{
		entries:    make(map[string]cacheEntry, maxEntries),
		ttl:        ttl,
		maxEntries: maxEntries,
	}
	return c
}

// cacheKey derives the cache key from contentVersion + mode + question.
// History is excluded on purpose (see the package comment). The
// contentVersion prefix means a prompt edit or knowledge update
// invalidates every existing entry on the next deploy — no manual cache
// flush required, no stale "old prompt" answers leaking past the TTL.
func cacheKey(mode Mode, question string) string {
	h := sha256.New()
	h.Write([]byte(contentVersion))
	h.Write([]byte{0})
	h.Write([]byte(mode))
	h.Write([]byte{0}) // separator so "modeQ" doesn't collide with "modQ"
	h.Write([]byte(question))
	return hex.EncodeToString(h.Sum(nil))
}

// Get returns the cached answer if present and not expired. ok is false on
// miss or expiry. Expired entries are not actively deleted here — the
// periodic sweep does that — but a stale entry never wins a Get because
// the createdAt check happens here.
func (c *answerCache) Get(key string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if time.Since(e.createdAt) > c.ttl {
		return "", false
	}
	return e.answer, true
}

// Set stores an answer under the given key. When the cache reaches its
// maxEntries cap, the oldest 25% of entries are dropped in one sweep —
// this amortises the cost of bookkeeping rather than running it on every
// write.
func (c *answerCache) Set(key, answer string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxEntries {
		c.evictOldestLocked()
	}
	c.entries[key] = cacheEntry{
		answer:    answer,
		createdAt: time.Now(),
	}
}

// evictOldestLocked drops expired entries plus, if still over budget, the
// oldest 25% by createdAt. The caller must hold the write lock.
func (c *answerCache) evictOldestLocked() {
	now := time.Now()
	// First pass — drop everything past TTL. Often this alone is enough.
	for k, e := range c.entries {
		if now.Sub(e.createdAt) > c.ttl {
			delete(c.entries, k)
		}
	}
	if len(c.entries) < c.maxEntries {
		return
	}
	// Second pass — sort remaining by createdAt, drop oldest 25%.
	type kv struct {
		key string
		at  time.Time
	}
	pairs := make([]kv, 0, len(c.entries))
	for k, e := range c.entries {
		pairs = append(pairs, kv{key: k, at: e.createdAt})
	}
	// O(n log n) insertion sort is fine — maxEntries is small (1000-ish).
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j].at.Before(pairs[j-1].at); j-- {
			pairs[j], pairs[j-1] = pairs[j-1], pairs[j]
		}
	}
	drop := len(pairs) / 4
	if drop == 0 {
		drop = 1
	}
	for _, p := range pairs[:drop] {
		delete(c.entries, p.key)
	}
}

// Stats returns the current entry count for logging / metrics.
func (c *answerCache) Stats() (entries int) {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
