package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_AllowWithinBurst(t *testing.T) {
	l := New(10, 5)
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatalf("expected allow on attempt %d", i)
		}
	}
	if l.Allow() {
		t.Fatal("expected deny after burst exhausted")
	}
}

func TestLimiter_Refills(t *testing.T) {
	l := New(100, 1)
	_ = l.Allow()
	time.Sleep(15 * time.Millisecond)
	if !l.Allow() {
		t.Fatal("expected allow after refill")
	}
}
