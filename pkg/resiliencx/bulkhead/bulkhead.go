// Package bulkhead provides concurrency limiting via semaphore.
package bulkhead

import (
	"context"
	"errors"
)

var ErrFull = errors.New("bulkhead is full")

// Bulkhead limits concurrent executions to maxConcurrent.
type Bulkhead struct {
	sem chan struct{}
}

// New creates a bulkhead that allows up to maxConcurrent concurrent calls.
func New(maxConcurrent int) *Bulkhead {
	return &Bulkhead{sem: make(chan struct{}, maxConcurrent)}
}

// Acquire blocks until a slot is available or ctx is cancelled.
func (b *Bulkhead) Acquire(ctx context.Context) error {
	select {
	case b.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire a slot without blocking.
func (b *Bulkhead) TryAcquire() error {
	select {
	case b.sem <- struct{}{}:
		return nil
	default:
		return ErrFull
	}
}

// Release returns a slot.
func (b *Bulkhead) Release() {
	<-b.sem
}

// Do acquires a slot, runs fn, and releases the slot.
func (b *Bulkhead) Do(ctx context.Context, fn func() error) error {
	if err := b.Acquire(ctx); err != nil {
		return err
	}
	defer b.Release()
	return fn()
}

// Available returns the number of currently available slots.
func (b *Bulkhead) Available() int {
	return cap(b.sem) - len(b.sem)
}
