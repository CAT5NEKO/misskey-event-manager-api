package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	general map[string]*rate.Limiter
	admin   map[string]*rate.Limiter
	mu      sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		general: make(map[string]*rate.Limiter),
		admin:   make(map[string]*rate.Limiter),
	}
	go rl.cleanupLoop()
	return rl
}

func isPrivateIP(ip string) bool {
	return strings.HasPrefix(ip, "127.") || strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "172.") ||
		ip == "::1" || ip == "localhost"
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.general[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(10), 50)
		rl.general[key] = limiter
	}
	return limiter
}

func (rl *RateLimiter) getAdminLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.admin[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(5), 30)
		rl.admin[key] = limiter
	}
	return limiter
}

func (rl *RateLimiter) GeneralLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetIPAddress(r)
		if isPrivateIP(ip) {
			next.ServeHTTP(w, r)
			return
		}
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			http.Error(w, `{"error":"リクエストが多すぎます。しばらく待ってください"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) AdminLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetIPAddress(r)
		if isPrivateIP(ip) {
			next.ServeHTTP(w, r)
			return
		}
		limiter := rl.getAdminLimiter(ip)
		if !limiter.Allow() {
			http.Error(w, `{"error":"リクエストが多すぎます。しばらく待ってください"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for key, limiter := range rl.general {
			if limiter.Tokens() >= float64(limiter.Burst()) {
				delete(rl.general, key)
			}
		}
		for key, limiter := range rl.admin {
			if limiter.Tokens() >= float64(limiter.Burst()) {
				delete(rl.admin, key)
			}
		}
		rl.mu.Unlock()
	}
}
