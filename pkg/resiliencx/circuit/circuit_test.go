package circuit

import (
	"errors"
	"testing"
	"time"
)

func TestBreaker_ClosedToOpen(t *testing.T) {
	b := New(3, time.Second)
	fail := errors.New("fail")
	for i := 0; i < 3; i++ {
		_ = b.Do(func() error { return fail })
	}
	if b.State() != Open {
		t.Fatalf("expected Open, got %v", b.State())
	}
}

func TestBreaker_OpenRejects(t *testing.T) {
	b := New(1, time.Hour)
	_ = b.Do(func() error { return errors.New("fail") })
	err := b.Do(func() error { return nil })
	if err != ErrOpen {
		t.Fatalf("expected ErrOpen, got %v", err)
	}
}

func TestBreaker_OpenToHalfOpen(t *testing.T) {
	b := New(1, 10*time.Millisecond)
	_ = b.Do(func() error { return errors.New("fail") })
	time.Sleep(20 * time.Millisecond)
	if b.State() != HalfOpen {
		t.Fatalf("expected HalfOpen, got %v", b.State())
	}
}

func TestBreaker_HalfOpen_ProbeSuccess(t *testing.T) {
	b := New(1, 10*time.Millisecond)
	_ = b.Do(func() error { return errors.New("fail") })
	time.Sleep(20 * time.Millisecond)
	err := b.Do(func() error { return nil })
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if b.State() != Closed {
		t.Fatalf("expected Closed, got %v", b.State())
	}
}

func TestBreaker_Reset(t *testing.T) {
	b := New(1, time.Hour)
	_ = b.Do(func() error { return errors.New("fail") })
	b.Reset()
	if b.State() != Closed {
		t.Fatalf("expected Closed after reset, got %v", b.State())
	}
}
