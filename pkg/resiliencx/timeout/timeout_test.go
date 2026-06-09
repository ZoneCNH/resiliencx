package timeout

import (
	"context"
	"testing"
	"time"
)

func TestDo_CompletesBeforeDeadline(t *testing.T) {
	err := Do(context.Background(), time.Second, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestDo_ExceedsDeadline(t *testing.T) {
	err := Do(context.Background(), 10*time.Millisecond, func(ctx context.Context) error {
		time.Sleep(time.Second)
		return nil
	})
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestDo_PropagatesError(t *testing.T) {
	want := context.Canceled
	err := Do(context.Background(), time.Second, func(ctx context.Context) error {
		return want
	})
	if err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}
