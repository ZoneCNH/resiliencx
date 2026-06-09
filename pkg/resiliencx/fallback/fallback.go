// Package fallback provides a primary-with-fallback execution pattern.
package fallback

import "context"

// Do tries fn first. If fn returns an error, it tries each fallback in order.
// Returns the first successful result, or the last error if all fail.
func Do(ctx context.Context, fn func(context.Context) error, fallbacks ...func(context.Context) error) error {
	if err := fn(ctx); err == nil {
		return nil
	} else if len(fallbacks) == 0 {
		return err
	}

	var lastErr error
	for _, fb := range fallbacks {
		if err := fb(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}
