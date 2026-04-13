package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerMinute,
		window:   time.Minute,
	}

	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, times := range rl.requests {
		validTimes := []time.Time{}
		for _, t := range times {
			if now.Sub(t) < rl.window {
				validTimes = append(validTimes, t)
			}
		}
		if len(validTimes) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validTimes
		}
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove old requests outside the window
	validTimes := []time.Time{}
	for _, t := range rl.requests[ip] {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}

	if len(validTimes) >= rl.limit {
		return false
	}

	rl.requests[ip] = append(validTimes, now)
	return true
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs separated by commas
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (proxy)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Get IP from remote address
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func AuthRateLimiter(limiter *RateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"too many requests, please try again later"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
