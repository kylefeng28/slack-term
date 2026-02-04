package service

import (
	"sync"
	"time"
)

type RateLimiter struct {
	tokens    int
	maxTokens int
	refillRate time.Duration
	mu        sync.Mutex
	lastRefill time.Time
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int(elapsed / r.refillRate)
	
	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = now
	}

	// Wait if no tokens available
	if r.tokens <= 0 {
		waitTime := r.refillRate
		r.mu.Unlock()
		time.Sleep(waitTime)
		r.mu.Lock()
		r.tokens = 1
		r.lastRefill = time.Now()
	}

	r.tokens--
}
