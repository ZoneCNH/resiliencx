package fallback

import (
	"context"
	"errors"
	"testing"
)

func TestDo_PrimarySucceeds(t *testing.T) {
	err := Do(context.Background(), func(ctx context.Context) error {
		return nil
	}, func(ctx context.Context) error {
		t.Fatal("fallback should not be called")
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestDo_FallbackSucceeds(t *testing.T) {
	err := Do(context.Background(), func(ctx context.Context) error {
		return errors.New("primary fail")
	}, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestDo_AllFail(t *testing.T) {
	last := errors.New("last fail")
	err := Do(context.Background(), func(ctx context.Context) error {
		return errors.New("primary")
	}, func(ctx context.Context) error {
		return errors.New("fb1")
	}, func(ctx context.Context) error {
		return last
	})
	if !errors.Is(err, last) {
		t.Fatalf("expected %v, got %v", last, err)
	}
}
