package resiliencx

import "errors"

// ErrAlreadyExecuted 表示该操作已经执行过。
var ErrAlreadyExecuted = errors.New("operation already executed")

// IdempotencyGuard 防止非幂等操作被自动重试。
type IdempotencyGuard struct {
	seen map[string]bool
}

// NewIdempotencyGuard 创建一个新的幂等守卫。
func NewIdempotencyGuard() *IdempotencyGuard {
	return &IdempotencyGuard{seen: make(map[string]bool)}
}

// Check 检查操作是否已执行过。
// 如果已执行过，返回 ErrAlreadyExecuted。
func (g *IdempotencyGuard) Check(key string) error {
	if g.seen[key] {
		return ErrAlreadyExecuted
	}
	return nil
}

// Mark 标记操作已执行。
func (g *IdempotencyGuard) Mark(key string) {
	g.seen[key] = true
}
