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

func TestBreaker_HalfOpen_ProbeFailure(t *testing.T) {
	b := New(1, 10*time.Millisecond)
	_ = b.Do(func() error { return errors.New("fail") })
	time.Sleep(20 * time.Millisecond)
	fail := errors.New("probe fail")
	err := b.Do(func() error { return fail })
	if !errors.Is(err, fail) {
		t.Fatalf("expected probe error, got %v", err)
	}
	if b.State() != Open {
		t.Fatalf("expected Open after failed probe, got %v", b.State())
	}
}

func TestBreaker_HalfOpen_SecondProbeRejected(t *testing.T) {
	b := New(1, 10*time.Millisecond)
	_ = b.Do(func() error { return errors.New("fail") })
	time.Sleep(20 * time.Millisecond)

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_ = b.Do(func() error {
			close(started)
			<-done
			return nil
		})
	}()
	<-started
	// Second probe while first is in flight
	err := b.Do(func() error { return nil })
	if err != ErrHalfOpen {
		t.Fatalf("expected ErrHalfOpen, got %v", err)
	}
	close(done)
}

func TestBreaker_ClosedResetsOnSuccess(t *testing.T) {
	b := New(3, time.Second)
	_ = b.Do(func() error { return errors.New("fail") })
	_ = b.Do(func() error { return errors.New("fail") })
	// Success resets failure count
	_ = b.Do(func() error { return nil })
	// Should still be closed
	if b.State() != Closed {
		t.Fatalf("expected Closed after success, got %v", b.State())
	}
	// Need 3 more failures to open
	for i := 0; i < 2; i++ {
		_ = b.Do(func() error { return errors.New("fail") })
	}
	if b.State() != Closed {
		t.Fatalf("expected Closed after 2 failures, got %v", b.State())
	}
}

func TestBreaker_State_ClosedFromOpen(t *testing.T) {
	b := New(1, 10*time.Millisecond)
	_ = b.Do(func() error { return errors.New("fail") })
	if b.State() != Open {
		t.Fatalf("expected Open, got %v", b.State())
	}
}
