# resiliencx 身份

## 我是谁

`resiliencx` 是 FoundationX 的 **L1 运行时弹性策略库**，为分布式系统中的不稳定外部依赖提供六种可组合的容错模式。

## 我做什么

| 策略 | 职责 |
|------|------|
| `timeout` | 函数调用超时控制 — PerAttemptTimeout + TotalTimeout |
| `retry` | 可配置指数退避重试 — max attempts / max elapsed / backoff / jitter |
| `circuit` | 三态熔断器 — Closed → Open → HalfOpen 状态机 |
| `bulkhead` | 信号量并发隔离 — 队列上限、快速拒绝 |
| `ratelimit` | 令牌桶限流 — QPS / burst / 按 key 限流 |
| `fallback` | 主逻辑 + 降级链 |

## 我不是什么

| 不是 | 原因 |
|------|------|
| **不是交易风控引擎** | 交易风控属于 `risk-engine` §CONSTITUTION P5 |
| **不是标准事实源** | 标准源属于 `xlib-standard` §CONSTITUTION P2 |
| **不是代码生成器** | Generator 职责属于 `xlib-standard` |
| **不承载业务域逻辑** | 交易/行情/风控逻辑属于分析域/决策域/执行域 |
| **不是 Harness Gate** | CI 门禁定义属于 `xlib-standard` |

## 我的边界

```
我拥有:
  - timeout / retry / circuit / bulkhead / ratelimit / fallback 策略实现
  - classifier（retryable / non-retryable / fatal 分类）
  - idempotency guard（非幂等操作禁止自动重试）
  - policy event sink（策略事件输出，交给 observex）
  - noop 安全默认（未配置时安全运行）
  - Option 模式配置

我不拥有:
  - 交易风控规则（属于 risk-engine）
  - 具体观测后端绑定（通过接口注入 observex）
  - 调度逻辑（属于 schedulex）
  - 配置解析（属于 configx）
  - 存储后端（redis/kafka/postgres 等）
```

## 我在架构中的位置

```
L0: kernel (stdlib-only primitives)
       ↑
L1: resiliencx ← 依赖 kernel（error/context/health），观测通过接口注入
       ↑
调用方: x.go / 业务模块 / schedulex adapter
```

## 与相似模块的区分

| 边界 | `kernel.retryx` | `resiliencx` |
|------|-----------------|-------------|
| 层级 | L0 primitive | L1 runtime policy |
| 职责 | backoff、retry marker、简单 retry loop | timeout、retry、circuit、bulkhead、rate、fallback |
| 观测 | 不负责完整 metrics | 输出 policy events，交给 observex 记录 |
| 状态 | 尽量无状态 | circuit breaker / limiter 可有状态 |
| 依赖 | stdlib only | 可依赖 kernel，观测通过接口注入 |
| 场景 | 基础库内部轻量重试 | 外部 API、交易所、数据源、消息、任务执行 |

## 宪法合规

| 条款 | 遵循方式 |
|------|----------|
| §1 P3 | `resiliencx` 只做运行时弹性，不做交易风控 |
| §1 P13 | 域内平级协作，策略包之间无编译期依赖 |
| §2.1 | 明确声明拥有/不拥有 |
| §3 | 仅依赖 kernel（L0），观测通过接口注入 |
| §5 | 测试覆盖率 100%（全 17 包） |
| §6 | 输出 policy events，交给 observex |
