package resiliencx

import (
	"context"
	"errors"
)

// RetryClass 错误的重试分类。
type RetryClass int

const (
	Retryable    RetryClass = iota // 可重试（临时性错误）
	NonRetryable                   // 不可重试（业务错误）
	Fatal                          // 致命（应立即终止）
)

// String 返回 RetryClass 的可读名称。
func (rc RetryClass) String() string {
	switch rc {
	case Retryable:
		return "retryable"
	case NonRetryable:
		return "non-retryable"
	case Fatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// Classifier 是错误分类函数，返回错误的重试分类。
type Classifier func(err error) RetryClass

// DefaultClassifier 返回默认错误分类器。
//
// 分类规则：
//   - context.Canceled → Fatal
//   - context.DeadlineExceeded → Retryable
//   - 其他错误 → NonRetryable
func DefaultClassifier() Classifier {
	return func(err error) RetryClass {
		if err == nil {
			return Retryable
		}
		if errors.Is(err, context.Canceled) {
			return Fatal
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return Retryable
		}
		return NonRetryable
	}
}
