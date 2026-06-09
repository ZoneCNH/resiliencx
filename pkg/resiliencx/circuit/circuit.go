// Package circuit provides a circuit breaker with three states:
// Closed (normal), Open (rejecting), HalfOpen (probing).
package circuit

import (
	"errors"
	"sync"
	"time"
)

// State represents the breaker state.
type State int

const (
	Closed   State = iota // normal operation
	Open                  // rejecting calls
	HalfOpen              // allowing a single probe
)

var (
	ErrOpen     = errors.New("circuit breaker is open")
	ErrHalfOpen = errors.New("circuit breaker is half-open, probe already in flight")
)

// Breaker implements the circuit breaker pattern.
type Breaker struct {
	mu            sync.Mutex
	state         State
	failures      int
	threshold     int
	cooldown      time.Duration
	lastFailTime  time.Time
	probeInFlight bool
}

// New creates a breaker that opens after threshold consecutive failures
// and transitions to half-open after cooldown.
func New(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{
		state:     Closed,
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// State returns the current breaker state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == Open && time.Since(b.lastFailTime) >= b.cooldown {
		b.state = HalfOpen
		b.probeInFlight = false
	}
	return b.state
}

// Do executes fn if the breaker allows it.
func (b *Breaker) Do(fn func() error) error {
	b.mu.Lock()
	if b.state == Open && time.Since(b.lastFailTime) >= b.cooldown {
		b.state = HalfOpen
		b.probeInFlight = false
	}

	switch b.state {
	case Open:
		b.mu.Unlock()
		return ErrOpen
	case HalfOpen:
		if b.probeInFlight {
			b.mu.Unlock()
			return ErrHalfOpen
		}
		b.probeInFlight = true
		b.mu.Unlock()

		err := fn()
		b.mu.Lock()
		b.probeInFlight = false
		if err != nil {
			b.trip()
		} else {
			b.state = Closed
			b.failures = 0
		}
		b.mu.Unlock()
		return err
	default: // Closed
		b.mu.Unlock()
		err := fn()
		b.mu.Lock()
		if err != nil {
			b.failures++
			if b.failures >= b.threshold {
				b.trip()
			}
		} else {
			b.failures = 0
		}
		b.mu.Unlock()
		return err
	}
}

// Reset forces the breaker back to Closed.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = Closed
	b.failures = 0
	b.probeInFlight = false
}

func (b *Breaker) trip() {
	b.state = Open
	b.lastFailTime = time.Now()
}
