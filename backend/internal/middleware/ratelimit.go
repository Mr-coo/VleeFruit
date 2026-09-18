package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// bucket is a token bucket refilled continuously at a fixed rate.
type bucket struct {
	tokens float64
	last   time.Time
}

// rateLimiter keeps one token bucket per client key (IP). It is safe for
// concurrent use and evicts idle buckets so memory does not grow unbounded.
type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rps      float64
	burst    float64
	lastSeen map[string]time.Time
}

// RateLimit returns middleware that throttles each client IP to rps sustained
// requests per second, allowing bursts up to burst. A non-positive rps disables
// limiting. Clients over the limit get 429 Too Many Requests.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	if rps <= 0 || burst <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	rl := &rateLimiter{
		buckets:  make(map[string]*bucket),
		lastSeen: make(map[string]time.Time),
		rps:      rps,
		burst:    float64(burst),
	}
	go rl.evictLoop()

	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			slog.WarnContext(c.Request.Context(), "rate limit exceeded",
				slog.String("client_ip", c.ClientIP()),
				slog.String("path", c.Request.URL.Path))
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// allow consumes one token for key, refilling based on elapsed time first.
func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, last: now}
		rl.buckets[key] = b
	} else {
		elapsed := now.Sub(b.last).Seconds()
		b.tokens = minFloat(rl.burst, b.tokens+elapsed*rl.rps)
		b.last = now
	}
	rl.lastSeen[key] = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// evictLoop periodically drops buckets that have been idle, bounding memory.
func (rl *rateLimiter) evictLoop() {
	const idle = 10 * time.Minute
	ticker := time.NewTicker(idle)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-idle)
		rl.mu.Lock()
		for key, seen := range rl.lastSeen {
			if seen.Before(cutoff) {
				delete(rl.buckets, key)
				delete(rl.lastSeen, key)
			}
		}
		rl.mu.Unlock()
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
