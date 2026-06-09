// Package ratelimit provides a token-bucket rate limiter.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter controls the rate of operations using a token bucket algorithm.
type Limiter struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	rate     float64
	lastTime time.Time
}

// New creates a limiter that fills at rate tokens/second up to max tokens.
func New(rate float64, max float64) *Limiter {
	return &Limiter{tokens: max, max: max, rate: rate, lastTime: time.Now()}
}

// Allow reports whether a single token is available now.
func (l *Limiter) Allow() bool { return l.AllowN(1) }

// AllowN reports whether n tokens are available now.
func (l *Limiter) AllowN(n float64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	if l.tokens >= n {
		l.tokens -= n
		return true
	}
	return false
}

// Reserve blocks until n tokens are available, returning the wait duration.
func (l *Limiter) Reserve(n float64) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	if l.tokens >= n {
		l.tokens -= n
		return 0
	}
	deficit := n - l.tokens
	wait := time.Duration(deficit/l.rate*1000) * time.Millisecond
	l.tokens = 0
	return wait
}

func (l *Limiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()
	l.lastTime = now
	l.tokens += elapsed * l.rate
	if l.tokens > l.max {
		l.tokens = l.max
	}
}
