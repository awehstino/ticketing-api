package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter holds the rate limiters for each IP address.
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// newIPRateLimiter creates a new IPRateLimiter.
func newIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

// addIP creates a new rate limiter for an IP address.
func (i *IPRateLimiter) addIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	limiter := rate.NewLimiter(i.r, i.b)
	i.ips[ip] = limiter
	return limiter
}

// getLimiter returns the rate limiter for a given IP address.
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()
	if !exists {
		return i.addIP(ip)
	}
	return limiter
}

// RateLimiter creates a middleware for rate limiting.
// It allows `limit` requests from a single IP over a given `duration`.
func RateLimiter(limit uint, duration time.Duration) gin.HandlerFunc {
	// Calculate rate as requests per second
	r := rate.Limit(float64(limit) / duration.Seconds())
	// Burst size is the same as the limit
	b := int(limit)
	limiter := newIPRateLimiter(r, b)

	return func(c *gin.Context) {
		ipLimiter := limiter.getLimiter(c.ClientIP())
		if !ipLimiter.Allow() {
			c.String(http.StatusTooManyRequests, "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}
