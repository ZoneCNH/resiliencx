package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SucceedsFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), DefaultPolicy(), func(ctx context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	p := Policy{MaxAttempts: 3, InitialWait: time.Millisecond, Multiplier: 1}
	err := Do(context.Background(), p, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("fail")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	want := errors.New("always fail")
	p := Policy{MaxAttempts: 2, InitialWait: time.Millisecond, Multiplier: 1}
	err := Do(context.Background(), p, func(ctx context.Context) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestDo_RespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := Policy{MaxAttempts: 5, InitialWait: time.Second, Multiplier: 1}
	err := Do(ctx, p, func(ctx context.Context) error {
		return errors.New("fail")
	})
	if err != context.Canceled {
		t.Fatalf("expected Canceled, got %v", err)
	}
}

func TestDo_DefaultMultiplier(t *testing.T) {
	calls := 0
	// Multiplier <= 0 should be reset to 2
	p := Policy{MaxAttempts: 2, InitialWait: time.Millisecond, Multiplier: 0}
	err := Do(context.Background(), p, func(ctx context.Context) error {
		calls++
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestDo_MaxWaitCap(t *testing.T) {
	calls := 0
	p := Policy{MaxAttempts: 4, InitialWait: time.Millisecond, MaxWait: 2 * time.Millisecond, Multiplier: 10}
	start := time.Now()
	err := Do(context.Background(), p, func(ctx context.Context) error {
		calls++
		return errors.New("fail")
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 4 {
		t.Fatalf("expected 4 calls, got %d", calls)
	}
	// With MaxWait cap, total wait should be bounded (3 sleeps, each capped at 2ms = ~6ms max + overhead)
	if elapsed > 100*time.Millisecond {
		t.Fatalf("expected bounded wait with MaxWait cap, took %v", elapsed)
	}
}

func TestDo_ContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	p := Policy{MaxAttempts: 100, InitialWait: 10 * time.Millisecond, Multiplier: 1}
	err := Do(ctx, p, func(ctx context.Context) error {
		return errors.New("fail")
	})
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}
