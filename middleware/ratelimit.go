package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	anonRate    = rate.Limit(5.0 / 3600.0)
	anonBurst   = 5
	authedRate  = rate.Limit(30.0 / 3600.0)
	authedBurst = 10

	cleanupInterval = 15 * time.Minute
	staleAfter      = 2 * time.Hour
)

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{buckets: map[string]*bucket{}}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, limit, burst := rl.classify(r)

		if !rl.allow(key, limit, burst) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) classify(r *http.Request) (key string, limit rate.Limit, burst int) {
	if uid := UserID(r.Context()); uid != "" {
		return "user:" + uid, authedRate, authedBurst
	}
	return "ip:" + clientIP(r), anonRate, anonBurst
}

func (rl *RateLimiter) allow(key string, limit rate.Limit, burst int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{limiter: rate.NewLimiter(limit, burst)}
		rl.buckets[key] = b
	}
	b.lastSeen = time.Now()
	return b.limiter.Allow()
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-staleAfter)
		rl.mu.Lock()
		for k, b := range rl.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// clientIP extracts the originating IP, preferring X-Real-IP, then the
// rightmost X-Forwarded-For entry (the one appended by the immediate
// trusted proxy), falling back to RemoteAddr. Assumes a single reverse
// proxy in front; if exposed directly, anon limits can be trivially
// bypassed by spoofing XFF.
func clientIP(r *http.Request) string {
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
