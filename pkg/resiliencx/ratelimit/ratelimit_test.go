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

func TestLimiter_AllowN_Insufficient(t *testing.T) {
	l := New(10, 3)
	if l.AllowN(4) {
		t.Fatal("expected deny when requesting more than max")
	}
}

func TestLimiter_AllowN_Exact(t *testing.T) {
	l := New(10, 5)
	if !l.AllowN(5) {
		t.Fatal("expected allow when requesting exactly max")
	}
	if l.AllowN(1) {
		t.Fatal("expected deny after consuming all tokens")
	}
}

func TestLimiter_Reserve_Immediate(t *testing.T) {
	l := New(10, 5)
	d := l.Reserve(3)
	if d != 0 {
		t.Fatalf("expected 0 duration, got %v", d)
	}
}

func TestLimiter_Reserve_Deficit(t *testing.T) {
	l := New(10, 5)
	// Consume all tokens
	l.AllowN(5)
	d := l.Reserve(3)
	if d <= 0 {
		t.Fatalf("expected positive duration for deficit, got %v", d)
	}
}

func TestLimiter_Reserve_PartialTokens(t *testing.T) {
	l := New(10, 5)
	l.AllowN(4)
	// 1 token left, request 3 => deficit 2, rate 10 => 200ms
	d := l.Reserve(3)
	if d <= 0 {
		t.Fatalf("expected positive duration, got %v", d)
	}
}

func TestLimiter_RefillCapsAtMax(t *testing.T) {
	l := New(1000, 5)
	time.Sleep(50 * time.Millisecond)
	// Should refill but cap at max=5
	if !l.AllowN(5) {
		t.Fatal("expected allow for max tokens after refill")
	}
	if l.AllowN(1) {
		t.Fatal("expected deny after consuming all capped tokens")
	}
}
