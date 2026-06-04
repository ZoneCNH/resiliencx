package resiliencx

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func retryableTestError() error {
	return WrapError(ErrorKindUnavailable, "test", "temporary", true, errors.New("boom"))
}

func TestRetryPolicyUsesBackoffClockAndClassifier(t *testing.T) {
	clock := NewManualClock(time.Unix(0, 0))
	policy := RetryPolicy{
		MaxAttempts: 3,
		Backoff: BackoffPolicy{
			Initial:    10 * time.Millisecond,
			Multiplier: 2,
			Jitter: func(attempt int, base time.Duration) time.Duration {
				return time.Duration(attempt) * time.Millisecond
			},
		},
		Clock: clock,
	}

	attempts := 0
	err := policy.Execute(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return retryableTestError()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retry execute: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	wantSleeps := []time.Duration{11 * time.Millisecond, 22 * time.Millisecond}
	if got := clock.Sleeps(); !reflect.DeepEqual(got, wantSleeps) {
		t.Fatalf("unexpected sleeps: got %v want %v", got, wantSleeps)
	}
}

func TestRetryPolicyStopsWhenContextCanceledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	clock := cancelingClock{cancel: cancel}
	policy := RetryPolicy{MaxAttempts: 2, Backoff: BackoffPolicy{Initial: time.Second}, Clock: clock}

	err := policy.Execute(ctx, func(context.Context) error { return retryableTestError() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestBulkheadAcquireRespectsContext(t *testing.T) {
	bulkhead := NewBulkhead(1)
	release, err := bulkhead.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := bulkhead.Acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimiterAcquireRespectsContext(t *testing.T) {
	clock := NewManualClock(time.Unix(0, 0))
	limiter := NewRateLimiter(1, 1, clock)
	if err := limiter.Acquire(context.Background()); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := limiter.Acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCircuitBreakerOpensAndHalfOpensWithClock(t *testing.T) {
	clock := NewManualClock(time.Unix(0, 0))
	breaker := NewCircuitBreaker(2, time.Second, clock)

	for i := 0; i < 2; i++ {
		if err := breaker.Execute(context.Background(), func(context.Context) error { return errors.New("fail") }); err == nil {
			t.Fatal("expected failure")
		}
	}
	if breaker.State() != CircuitOpen {
		t.Fatalf("expected open, got %s", breaker.State())
	}
	if err := breaker.Execute(context.Background(), func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected open circuit error")
	}
	clock.Advance(time.Second)
	if breaker.State() != CircuitHalfOpen {
		t.Fatalf("expected half-open, got %s", breaker.State())
	}
	if err := breaker.Execute(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("half-open success: %v", err)
	}
	if breaker.State() != CircuitClosed {
		t.Fatalf("expected closed, got %s", breaker.State())
	}
}

func TestPolicyComposesPrimitivesAndHooksArePanicSafe(t *testing.T) {
	clock := NewManualClock(time.Unix(0, 0))
	calls := 0
	var observed Event
	policy := Policy{
		Retry:       &RetryPolicy{MaxAttempts: 2, Backoff: BackoffPolicy{Initial: time.Millisecond}, Clock: clock},
		Bulkhead:    NewBulkhead(1),
		RateLimiter: NewRateLimiter(100, 1, clock),
		Breaker:     NewCircuitBreaker(3, time.Second, clock),
		Hooks: []Hook{
			func(Event) { panic("hook panic must not escape") },
			func(event Event) { observed = event },
		},
	}

	err := policy.Execute(context.Background(), "policy.test", func(context.Context) error {
		calls++
		if calls == 1 {
			return retryableTestError()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("policy execute: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if observed.Name != "policy.test" || observed.Error != nil {
		t.Fatalf("unexpected event: %+v", observed)
	}
}

type cancelingClock struct{ cancel context.CancelFunc }

func (c cancelingClock) Now() time.Time { return time.Unix(0, 0) }

func (c cancelingClock) Sleep(ctx context.Context, _ time.Duration) error {
	c.cancel()
	return ctx.Err()
}
