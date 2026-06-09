package resiliencx

import (
	"context"
	"errors"
	"testing"
)

func TestNoopStrategy_Do(t *testing.T) {
	s := NoopStrategy{}
	called := false
	err := s.Do(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})
	if !called {
		t.Error("fn was not called")
	}
	if err != nil {
		t.Errorf("Do returned error: %v", err)
	}
}

func TestNoopStrategy_Do_PropagatesError(t *testing.T) {
	s := NoopStrategy{}
	want := errors.New("fail")
	err := s.Do(context.Background(), func(ctx context.Context) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Errorf("Do error = %v, want %v", err, want)
	}
}

func TestNoopStrategy_Name(t *testing.T) {
	s := NoopStrategy{}
	if got := s.Name(); got != "noop" {
		t.Errorf("Name() = %q, want %q", got, "noop")
	}
}

func TestNoopStrategy_Do_IgnoresPassedContext(t *testing.T) {
	s := NoopStrategy{}
	// Pass a cancelled context; NoopStrategy should use context.Background() internally.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var gotCtx context.Context
	_ = s.Do(ctx, func(ctx context.Context) error {
		gotCtx = ctx
		return nil
	})
	if gotCtx.Err() != nil {
		t.Error("NoopStrategy should use context.Background(), not the passed context")
	}
}
