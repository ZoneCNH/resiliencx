package bulkhead

import (
	"context"
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
