package resiliencx

import "time"

// EventType 策略事件类型。
type EventType int

const (
	EventRetry           EventType = iota // 重试
	EventCircuitOpen                      // 熔断器打开
	EventCircuitClose                     // 熔断器关闭
	EventBulkheadReject                   // 舱壁拒绝
	EventRateLimitReject                  // 限流拒绝
	EventTimeout                          // 超时
	EventFallback                         // 降级
)

// String 返回 EventType 的可读名称。
func (et EventType) String() string {
	switch et {
	case EventRetry:
		return "retry"
	case EventCircuitOpen:
		return "circuit_open"
	case EventCircuitClose:
		return "circuit_close"
	case EventBulkheadReject:
		return "bulkhead_reject"
	case EventRateLimitReject:
		return "rate_limit_reject"
	case EventTimeout:
		return "timeout"
	case EventFallback:
		return "fallback"
	default:
		return "unknown"
	}
}

// Event 策略事件。
type Event struct {
	Type     EventType
	Time     time.Time
	Attempt  int
	Err      error
	Duration time.Duration
	Metadata map[string]any
}

// Sink 事件接收器接口。
type Sink interface {
	Emit(event Event)
}

// NoopSink 空实现，丢弃所有事件。
type NoopSink struct{}

func (NoopSink) Emit(Event) {}

// SliceSink 用于测试，将事件收集到 slice 中。
type SliceSink struct {
	Events []Event
}

func (s *SliceSink) Emit(e Event) {
	s.Events = append(s.Events, e)
}
