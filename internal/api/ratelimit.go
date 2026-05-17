package api

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rateLimiter is a lightweight in-memory token-bucket limiter keyed by client.
// No Redis, no external state — buckets live in a map guarded by a mutex and
// are swept periodically by the server's janitor so the map cannot grow
// unbounded. This is plenty for a single-instance Railway deployment.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*bucket
	rate     float64 // tokens refilled per second
	burst    float64 // maximum tokens (also the per-minute allowance)
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// newRateLimiter builds a limiter allowing roughly perMinute requests per key,
// with a burst capacity equal to perMinute.
func newRateLimiter(perMinute int) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string]*bucket),
		rate:     float64(perMinute) / 60.0,
		burst:    float64(perMinute),
	}
}

// allow consumes one token for key, returning false when the bucket is empty.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.visitors[key]
	if !ok {
		rl.visitors[key] = &bucket{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	// Refill proportionally to elapsed time, capped at burst.
	b.tokens = math.Min(rl.burst, b.tokens+now.Sub(b.lastSeen).Seconds()*rl.rate)
	b.lastSeen = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// sweep removes buckets idle longer than maxIdle. Called by the server janitor.
func (rl *rateLimiter) sweep(maxIdle time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for key, b := range rl.visitors {
		if time.Since(b.lastSeen) > maxIdle {
			delete(rl.visitors, key)
		}
	}
}

// rateLimit returns a middleware that enforces rl, keyed by client IP.
func (s *Server) rateLimit(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(clientIP(r)) {
				WriteError(w, http.StatusTooManyRequests, "слишком много запросов, попробуй позже")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitByAdmin returns a middleware that enforces rl keyed by the
// authenticated admin's user ID rather than client IP. It must run AFTER
// requireTelegramAuth + requireAdmin, so the user is already in the context.
//
// Keying by identity (not IP) hardens the most sensitive mutating endpoint:
// behind Railway's proxy many admins could share an egress IP, and an IP-only
// limit is both too coarse and trivially sidestepped by rotating IPs. A
// per-admin cap throttles each operator's write rate deterministically.
func (s *Server) rateLimitByAdmin(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "anon"
			if u := userFromContext(r.Context()); u != nil {
				key = "admin:" + strconv.FormatInt(u.ID, 10)
			}
			if !rl.allow(key) {
				WriteError(w, http.StatusTooManyRequests, "слишком много корректировок, попробуй позже")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the caller's IP for rate-limiting.
//
// SECURITY: X-Forwarded-For is partly client-controlled. A client may PREPEND
// arbitrary entries, but it cannot append past the entry the trusted proxy
// adds — so the RIGHTMOST entry is the address that actually connected to
// Railway's edge proxy (the real client) and is not spoofable. Reading the
// leftmost entry instead would let any caller mint a fresh rate-limit bucket
// per request simply by rotating a header, defeating rate limiting entirely.
//
// This assumes exactly one trusted proxy hop (Railway's edge) in front of the
// container, which is the deployment topology.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
			return last
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
