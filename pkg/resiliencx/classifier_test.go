package resiliencx

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultClassifier_ContextCanceled(t *testing.T) {
	c := DefaultClassifier()
	if got := c(context.Canceled); got != Fatal {
		t.Errorf("context.Canceled: got %v, want Fatal", got)
	}
}

func TestDefaultClassifier_ContextDeadlineExceeded(t *testing.T) {
	c := DefaultClassifier()
	if got := c(context.DeadlineExceeded); got != Retryable {
		t.Errorf("context.DeadlineExceeded: got %v, want Retryable", got)
	}
}

func TestDefaultClassifier_OtherError(t *testing.T) {
	c := DefaultClassifier()
	err := errors.New("some business error")
	if got := c(err); got != NonRetryable {
		t.Errorf("other error: got %v, want NonRetryable", got)
	}
}

func TestDefaultClassifier_NilError(t *testing.T) {
	c := DefaultClassifier()
	if got := c(nil); got != Retryable {
		t.Errorf("nil error: got %v, want Retryable", got)
	}
}

func TestDefaultClassifier_WrappedCanceled(t *testing.T) {
	c := DefaultClassifier()
	wrapped := errors.Join(errors.New("outer"), context.Canceled)
	if got := c(wrapped); got != Fatal {
		t.Errorf("wrapped Canceled: got %v, want Fatal", got)
	}
}

func TestDefaultClassifier_WrappedDeadlineExceeded(t *testing.T) {
	c := DefaultClassifier()
	wrapped := errors.Join(errors.New("outer"), context.DeadlineExceeded)
	if got := c(wrapped); got != Retryable {
		t.Errorf("wrapped DeadlineExceeded: got %v, want Retryable", got)
	}
}

func TestRetryClass_String(t *testing.T) {
	tests := []struct {
		class RetryClass
		want  string
	}{
		{Retryable, "retryable"},
		{NonRetryable, "non-retryable"},
		{Fatal, "fatal"},
		{RetryClass(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.class.String(); got != tt.want {
			t.Errorf("RetryClass(%d).String() = %q, want %q", tt.class, got, tt.want)
		}
	}
}
