// Package timeout provides deadline enforcement for function calls.
package timeout

import (
	"context"
	"time"
)

// Do runs fn within the given duration. It returns context.DeadlineExceeded
// if fn does not complete in time.
func Do(ctx context.Context, d time.Duration, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	ch := make(chan error, 1)
	go func() { ch <- fn(ctx) }()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
