// Package resiliencx provides context-aware resilience primitives for L1 libraries.
//
// The package intentionally stays vendor-neutral: callers compose retry, timeout,
// backoff, jitter, circuit breaker, bulkhead, rate-limit, and failure-budget
// policies without importing provider, L2, x.go, or observability implementations.
// All waits and resource acquisitions accept context.Context, and time is injected
// through Clock so behavior is deterministic in tests.
package resiliencx
