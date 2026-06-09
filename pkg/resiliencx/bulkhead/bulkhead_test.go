package bulkhead

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkhead_LimitsConcurrency(t *testing.T) {
	b := New(2)
	var running atomic.Int32
	var maxRunning atomic.Int32

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Do(context.Background(), func() error {
				cur := running.Add(1)
				for {
					old := maxRunning.Load()
					if cur <= old || maxRunning.CompareAndSwap(old, cur) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				running.Add(-1)
				return nil
			})
		}()
	}
	wg.Wait()
	if maxRunning.Load() > 2 {
		t.Fatalf("max concurrent %d, expected <= 2", maxRunning.Load())
	}
}

func TestBulkhead_TryAcquire_Full(t *testing.T) {
	b := New(1)
	_ = b.Acquire(context.Background())
	err := b.TryAcquire()
	if err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}
}

func TestBulkhead_Available(t *testing.T) {
	b := New(3)
	if b.Available() != 3 {
		t.Fatalf("expected 3, got %d", b.Available())
	}
}

func TestBulkhead_Acquire_CtxCancelled(t *testing.T) {
	b := New(1)
	_ = b.Acquire(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := b.Acquire(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestBulkhead_TryAcquire_Success(t *testing.T) {
	b := New(2)
	if err := b.TryAcquire(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := b.TryAcquire(); err != nil {
		t.Fatalf("expected nil on second acquire, got %v", err)
	}
	if err := b.TryAcquire(); err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}
}

func TestBulkhead_Do_CtxCancelled(t *testing.T) {
	b := New(1)
	_ = b.Acquire(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := b.Do(ctx, func() error { return nil })
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestBulkhead_Do_FnError(t *testing.T) {
	b := New(1)
	want := errors.New("fn error")
	err := b.Do(context.Background(), func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	// Slot should be released after fn error
	if b.Available() != 1 {
		t.Fatalf("expected 1 available after release, got %d", b.Available())
	}
}

func TestBulkhead_Available_AfterAcquire(t *testing.T) {
	b := New(3)
	_ = b.TryAcquire()
	if b.Available() != 2 {
		t.Fatalf("expected 2, got %d", b.Available())
	}
	b.Release()
	if b.Available() != 3 {
		t.Fatalf("expected 3 after release, got %d", b.Available())
	}
}
