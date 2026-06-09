package resiliencx

import (
	"errors"
	"testing"
	"time"
)

func TestNoopSink_Emit(t *testing.T) {
	var s NoopSink
	// Should not panic.
	s.Emit(Event{Type: EventRetry, Time: time.Now()})
}

func TestSliceSink_CollectsEvents(t *testing.T) {
	s := &SliceSink{}
	now := time.Now()

	e1 := Event{Type: EventRetry, Time: now, Attempt: 1}
	e2 := Event{Type: EventCircuitOpen, Time: now.Add(time.Second), Err: errors.New("open")}
	e3 := Event{Type: EventTimeout, Time: now.Add(2 * time.Second), Duration: 5 * time.Second}

	s.Emit(e1)
	s.Emit(e2)
	s.Emit(e3)

	if len(s.Events) != 3 {
		t.Fatalf("len(Events) = %d, want 3", len(s.Events))
	}
	if s.Events[0].Type != EventRetry {
		t.Errorf("Events[0].Type = %v, want EventRetry", s.Events[0].Type)
	}
	if s.Events[1].Type != EventCircuitOpen {
		t.Errorf("Events[1].Type = %v, want EventCircuitOpen", s.Events[1].Type)
	}
	if s.Events[2].Duration != 5*time.Second {
		t.Errorf("Events[2].Duration = %v, want 5s", s.Events[2].Duration)
	}
}

func TestSliceSink_EmptyByDefault(t *testing.T) {
	s := &SliceSink{}
	if len(s.Events) != 0 {
		t.Errorf("empty SliceSink has %d events, want 0", len(s.Events))
	}
}

func TestEventType_String(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventRetry, "retry"},
		{EventCircuitOpen, "circuit_open"},
		{EventCircuitClose, "circuit_close"},
		{EventBulkheadReject, "bulkhead_reject"},
		{EventRateLimitReject, "rate_limit_reject"},
		{EventTimeout, "timeout"},
		{EventFallback, "fallback"},
		{EventType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.et.String(); got != tt.want {
			t.Errorf("EventType(%d).String() = %q, want %q", tt.et, got, tt.want)
		}
	}
}
