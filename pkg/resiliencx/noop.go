package resiliencx

import "context"

// NoopStrategy 未配置时的安全默认值。
// 直接执行 fn，不做任何策略包装。
type NoopStrategy struct{}

func (NoopStrategy) Do(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}

func (NoopStrategy) Name() string { return "noop" }
