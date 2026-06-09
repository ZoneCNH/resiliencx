# resiliencx

运行时弹性策略库，为分布式系统提供六种可组合的容错模式。

## 策略

| 包 | 功能 | 入口 |
|---|------|------|
| `timeout` | 函数调用超时控制 | `timeout.Do(ctx, duration, fn)` |
| `retry` | 可配置指数退避重试 | `retry.Do(ctx, policy, fn)` |
| `circuit` | 三态熔断器 (Closed/Open/HalfOpen) | `circuit.New(threshold, cooldown)` |
| `bulkhead` | 信号量并发限制 | `bulkhead.New(maxConcurrent)` |
| `ratelimit` | 令牌桶限流 | `ratelimit.New(rate, max)` |
| `fallback` | 主逻辑 + 降级链 | `fallback.Do(ctx, fn, fallbacks...)` |

## 快速开始

```go
import "github.com/ZoneCNH/resiliencx/pkg/resiliencx/retry"

err := retry.Do(ctx, retry.DefaultPolicy(), func(ctx context.Context) error {
    return callExternalService(ctx)
})
```

```go
import "github.com/ZoneCNH/resiliencx/pkg/resiliencx/circuit"

breaker := circuit.New(5, 30*time.Second) // 5 次失败后熔断，30 秒后半开
err := breaker.Do(func() error {
    return callExternalService()
})
if errors.Is(err, circuit.ErrOpen) {
    // 降级处理
}
```

## 设计原则

- 每个策略独立子包，按需导入
- 接受 `context.Context`，尊重取消和超时
- 无外部依赖，仅使用标准库
- 可组合：策略之间可以嵌套使用

## 测试

```bash
go test ./pkg/resiliencx/...
```

## 目录结构

```
pkg/resiliencx/
├── timeout/     # 超时控制
├── retry/       # 重试策略
├── circuit/     # 熔断器
├── bulkhead/    # 并发限制
├── ratelimit/   # 限流器
└── fallback/    # 降级链
```

## 非目标

- 不提供服务发现或负载均衡
- 不提供分布式锁或分布式限流（需 Redis/NATS 等外部组件）
- 不包含业务逻辑或策略参数的自动调优
- 不依赖 `x.go` 或任何业务域模块

## 关联模块

| 模块 | 关系 |
|------|------|
| [kernel](https://github.com/ZoneCNH/kernel) | 提供 error/context/lifecycle 原语 |
| [configx](https://github.com/ZoneCNH/configx) | 提供策略参数配置加载 |
| [observex](https://github.com/ZoneCNH/observex) | 提供弹性事件 metrics/tracing |
| [order-engine](https://github.com/ZoneCNH/order-engine) | 消费方：交易所调用容错 |
| [market-data](https://github.com/ZoneCNH/market-data) | 消费方：数据采集容错 |
