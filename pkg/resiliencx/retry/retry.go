// Package retry provides configurable retry with backoff.
package retry

import (
	"context"
	"time"
)

// Policy controls retry behaviour.
type Policy struct {
	MaxAttempts int           // total attempts (1 = no retry)
	InitialWait time.Duration // first backoff interval
	MaxWait     time.Duration // cap on backoff
	Multiplier  float64       // backoff multiplier (default 2)
}

// DefaultPolicy returns a sensible default: 3 attempts, 100ms initial, 5s max, 2x multiplier.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  2,
	}
}

// Do executes fn up to p.MaxAttempts times, sleeping between retries with
// exponential backoff. Returns the first non-nil error after all attempts
// are exhausted, or nil if any attempt succeeds.
func Do(ctx context.Context, p Policy, fn func(context.Context) error) error {
	if p.Multiplier <= 0 {
		p.Multiplier = 2
	}

	wait := p.InitialWait
	var lastErr error

	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if err := fn(ctx); err != nil {
			lastErr = err
		} else {
			return nil
		}

		if attempt == p.MaxAttempts-1 {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		wait = time.Duration(float64(wait) * p.Multiplier)
		if p.MaxWait > 0 && wait > p.MaxWait {
			wait = p.MaxWait
		}
	}

	return lastErr
}
