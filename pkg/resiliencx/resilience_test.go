package resiliencx

import (
	"context"
	"testing"
)

func TestNewResilienceConfig_Defaults(t *testing.T) {
	cfg := NewResilienceConfig()
	if cfg.Classifier == nil {
		t.Error("default Classifier is nil")
	}
	if cfg.Sink == nil {
		t.Error("default Sink is nil")
	}
	// Verify default classifier works.
	if got := cfg.Classifier(context.Canceled); got != Fatal {
		t.Errorf("default Classifier(Canceled) = %v, want Fatal", got)
	}
}

func TestWithClassifier(t *testing.T) {
	custom := func(err error) RetryClass { return Retryable }
	cfg := NewResilienceConfig(WithClassifier(custom))
	if cfg.Classifier == nil {
		t.Error("WithClassifier: Classifier is nil")
	}
	// Custom classifier always returns Retryable.
	if got := cfg.Classifier(context.Canceled); got != Retryable {
		t.Errorf("custom Classifier(Canceled) = %v, want Retryable", got)
	}
}

func TestWithSink(t *testing.T) {
	sink := &SliceSink{}
	cfg := NewResilienceConfig(WithSink(sink))
	if cfg.Sink == nil {
		t.Error("WithSink: Sink is nil")
	}
	cfg.Sink.Emit(Event{Type: EventRetry})
	if len(sink.Events) != 1 {
		t.Errorf("SliceSink received %d events, want 1", len(sink.Events))
	}
}

func TestMultipleOptions(t *testing.T) {
	sink := &SliceSink{}
	classifier := func(err error) RetryClass { return Fatal }
	cfg := NewResilienceConfig(
		WithClassifier(classifier),
		WithSink(sink),
	)
	if cfg.Classifier == nil || cfg.Sink == nil {
		t.Error("multiple options: one or more fields are nil")
	}
}
