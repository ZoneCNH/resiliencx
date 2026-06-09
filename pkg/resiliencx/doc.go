// Package resiliencx provides runtime resilience strategies for distributed systems.
//
// It offers six composable patterns:
//   - timeout: deadline enforcement for function calls
//   - retry: configurable retry with exponential backoff
//   - circuit: circuit breaker (Closed/Open/HalfOpen)
//   - bulkhead: concurrency limiting via semaphore
//   - ratelimit: token-bucket rate limiter
//   - fallback: primary-with-fallback execution
//
// Each pattern is available as a standalone sub-package under resiliencx/.
// This package must not depend on github.com/ZoneCNH/x.go internals.
package resiliencx
