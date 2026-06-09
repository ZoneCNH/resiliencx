package resiliencx

import (
	"errors"
	"testing"
)

func TestIdempotencyGuard_CheckNewKey(t *testing.T) {
	g := NewIdempotencyGuard()
	if err := g.Check("key-1"); err != nil {
		t.Errorf("Check new key: got %v, want nil", err)
	}
}

func TestIdempotencyGuard_MarkThenCheck(t *testing.T) {
	g := NewIdempotencyGuard()
	g.Mark("key-1")
	if err := g.Check("key-1"); !errors.Is(err, ErrAlreadyExecuted) {
		t.Errorf("Check after Mark: got %v, want ErrAlreadyExecuted", err)
	}
}

func TestIdempotencyGuard_DifferentKeys(t *testing.T) {
	g := NewIdempotencyGuard()
	g.Mark("key-1")
	if err := g.Check("key-2"); err != nil {
		t.Errorf("Check different key: got %v, want nil", err)
	}
}

func TestIdempotencyGuard_MarkTwiceIdempotent(t *testing.T) {
	g := NewIdempotencyGuard()
	g.Mark("key-1")
	g.Mark("key-1") // should not panic
	if err := g.Check("key-1"); !errors.Is(err, ErrAlreadyExecuted) {
		t.Errorf("Check after double Mark: got %v, want ErrAlreadyExecuted", err)
	}
}

func TestErrAlreadyExecuted(t *testing.T) {
	if ErrAlreadyExecuted.Error() != "operation already executed" {
		t.Errorf("ErrAlreadyExecuted.Error() = %q", ErrAlreadyExecuted.Error())
	}
}
