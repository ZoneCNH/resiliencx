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
