package resiliencx

import (
	"context"
	"sync"
	"time"
)

// Clock abstracts time so resilience policies are deterministic in tests.
type Clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}

type realClock struct{}

// RealClock returns a clock backed by time.Now and context-aware timers.
func RealClock() Clock { return realClock{} }

func (realClock) Now() time.Time { return time.Now() }

func (realClock) Sleep(ctx context.Context, d time.Duration) error {
	if ctx == nil {
		return validationError("Clock.Sleep", "context is required", nil)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return ctx.Err()
	}
}

// ManualClock is a deterministic Clock for tests and simulations.
type ManualClock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
}

// NewManualClock returns a manual clock pinned to start.
func NewManualClock(start time.Time) *ManualClock { return &ManualClock{now: start} }

// Now returns the current manual time.
func (c *ManualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves manual time forward. Non-positive durations are ignored.
func (c *ManualClock) Advance(d time.Duration) {
	if d <= 0 {
		return
	}
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// Sleep records the wait, advances manual time, and still honors context cancellation.
func (c *ManualClock) Sleep(ctx context.Context, d time.Duration) error {
	if ctx == nil {
		return validationError("ManualClock.Sleep", "context is required", nil)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return ctx.Err()
	}
	c.mu.Lock()
	c.sleeps = append(c.sleeps, d)
	c.now = c.now.Add(d)
	c.mu.Unlock()
	return ctx.Err()
}

// Sleeps returns a copy of durations passed to Sleep.
func (c *ManualClock) Sleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]time.Duration, len(c.sleeps))
	copy(out, c.sleeps)
	return out
}
