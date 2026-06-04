package resiliencx

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"
)

// Operation is the work protected by a resilience policy.
type Operation func(context.Context) error

type Decision int

const (
	DecisionRetry Decision = iota
	DecisionStop
)

type Classifier func(error) Decision

func DefaultClassifier(err error) Decision {
	if err == nil || errors.Is(err, context.Canceled) {
		return DecisionStop
	}
	var e *Error
	if errors.As(err, &e) && e.Retryable {
		return DecisionRetry
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return DecisionRetry
	}
	return DecisionStop
}

type JitterFunc func(attempt int, base time.Duration) time.Duration

type BackoffPolicy struct {
	Initial    time.Duration
	Multiplier float64
	Max        time.Duration
	Jitter     JitterFunc
}

func (p BackoffPolicy) Delay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	initial := p.Initial
	if initial <= 0 {
		initial = time.Millisecond
	}
	multiplier := p.Multiplier
	if multiplier < 1 {
		multiplier = 1
	}
	d := float64(initial)
	if attempt > 1 {
		d *= math.Pow(multiplier, float64(attempt-1))
	}
	if max := p.Max; max > 0 && d > float64(max) {
		d = float64(max)
	}
	delay := time.Duration(d)
	if p.Jitter != nil {
		delay += p.Jitter(attempt, delay)
		if delay < 0 {
			return 0
		}
	}
	return delay
}

type RetryPolicy struct {
	MaxAttempts int
	Backoff     BackoffPolicy
	Classifier  Classifier
	Clock       Clock
}

func (p RetryPolicy) Execute(ctx context.Context, op Operation) error {
	if ctx == nil {
		return validationError("Retry.Execute", "context is required", nil)
	}
	if op == nil {
		return validationError("Retry.Execute", "operation is required", nil)
	}
	clock := p.Clock
	if clock == nil {
		clock = RealClock()
	}
	classify := p.Classifier
	if classify == nil {
		classify = DefaultClassifier
	}
	maxAttempts := p.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	var last error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = op(ctx)
		if last == nil {
			return nil
		}
		if attempt == maxAttempts || classify(last) != DecisionRetry {
			return last
		}
		if err := clock.Sleep(ctx, p.Backoff.Delay(attempt)); err != nil {
			return err
		}
	}
	return last
}

type TimeoutPolicy struct{ Duration time.Duration }

func (p TimeoutPolicy) Execute(ctx context.Context, op Operation) error {
	if ctx == nil {
		return validationError("Timeout.Execute", "context is required", nil)
	}
	if op == nil {
		return validationError("Timeout.Execute", "operation is required", nil)
	}
	if p.Duration <= 0 {
		return op(ctx)
	}
	child, cancel := context.WithTimeout(ctx, p.Duration)
	defer cancel()
	return op(child)
}

type Bulkhead struct{ slots chan struct{} }

func NewBulkhead(limit int) *Bulkhead {
	if limit <= 0 {
		limit = 1
	}
	return &Bulkhead{slots: make(chan struct{}, limit)}
}

func (b *Bulkhead) Acquire(ctx context.Context) (func(), error) {
	if ctx == nil {
		return nil, validationError("Bulkhead.Acquire", "context is required", nil)
	}
	if b == nil || b.slots == nil {
		return nil, validationError("Bulkhead.Acquire", "bulkhead is not initialized", nil)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case b.slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-b.slots }) }, nil
	}
}

func (b *Bulkhead) Execute(ctx context.Context, op Operation) error {
	if op == nil {
		return validationError("Bulkhead.Execute", "operation is required", nil)
	}
	release, err := b.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	return op(ctx)
}

type RateLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
	clock  Clock
}

func NewRateLimiter(ratePerSecond float64, burst int, clock Clock) *RateLimiter {
	if ratePerSecond <= 0 {
		ratePerSecond = 1
	}
	if burst <= 0 {
		burst = 1
	}
	if clock == nil {
		clock = RealClock()
	}
	now := clock.Now()
	return &RateLimiter{rate: ratePerSecond, burst: float64(burst), tokens: float64(burst), last: now, clock: clock}
}

func (r *RateLimiter) Acquire(ctx context.Context) error {
	if ctx == nil {
		return validationError("RateLimiter.Acquire", "context is required", nil)
	}
	if r == nil {
		return validationError("RateLimiter.Acquire", "limiter is not initialized", nil)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.mu.Lock()
		now := r.clock.Now()
		if elapsed := now.Sub(r.last).Seconds(); elapsed > 0 {
			r.tokens = math.Min(r.burst, r.tokens+elapsed*r.rate)
			r.last = now
		}
		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}
		missing := 1 - r.tokens
		wait := time.Duration(math.Ceil((missing / r.rate) * float64(time.Second)))
		r.mu.Unlock()
		if wait <= 0 {
			wait = time.Nanosecond
		}
		if err := r.clock.Sleep(ctx, wait); err != nil {
			return err
		}
	}
}

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

type CircuitBreaker struct {
	mu               sync.Mutex
	failureThreshold int
	openFor          time.Duration
	clock            Clock
	state            CircuitState
	failures         int
	openedAt         time.Time
}

func NewCircuitBreaker(failureThreshold int, openFor time.Duration, clock Clock) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 1
	}
	if clock == nil {
		clock = RealClock()
	}
	return &CircuitBreaker{failureThreshold: failureThreshold, openFor: openFor, clock: clock, state: CircuitClosed}
}

func (b *CircuitBreaker) State() CircuitState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentStateLocked()
}

func (b *CircuitBreaker) Execute(ctx context.Context, op Operation) error {
	if ctx == nil {
		return validationError("CircuitBreaker.Execute", "context is required", nil)
	}
	if op == nil {
		return validationError("CircuitBreaker.Execute", "operation is required", nil)
	}
	if b == nil {
		return validationError("CircuitBreaker.Execute", "breaker is not initialized", nil)
	}
	b.mu.Lock()
	state := b.currentStateLocked()
	if state == CircuitOpen {
		b.mu.Unlock()
		return NewError(ErrorKindUnavailable, "CircuitBreaker.Execute", "circuit is open", true)
	}
	b.mu.Unlock()

	err := op(ctx)
	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		b.failures = 0
		b.state = CircuitClosed
		return nil
	}
	b.failures++
	if b.failures >= b.failureThreshold {
		b.state = CircuitOpen
		b.openedAt = b.clock.Now()
	}
	return err
}

func (b *CircuitBreaker) currentStateLocked() CircuitState {
	if b.state == CircuitOpen && b.openFor > 0 && !b.clock.Now().Before(b.openedAt.Add(b.openFor)) {
		b.state = CircuitHalfOpen
	}
	return b.state
}

type FailureBudget struct {
	mu          sync.Mutex
	maxFailures int
	window      time.Duration
	clock       Clock
	start       time.Time
	failures    int
}

func NewFailureBudget(maxFailures int, window time.Duration, clock Clock) *FailureBudget {
	if maxFailures <= 0 {
		maxFailures = 1
	}
	if clock == nil {
		clock = RealClock()
	}
	return &FailureBudget{maxFailures: maxFailures, window: window, clock: clock, start: clock.Now()}
}

func (b *FailureBudget) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resetIfExpiredLocked()
	return b.failures < b.maxFailures
}

func (b *FailureBudget) Record(err error) {
	if err == nil || b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resetIfExpiredLocked()
	b.failures++
}

func (b *FailureBudget) resetIfExpiredLocked() {
	if b.window > 0 && !b.clock.Now().Before(b.start.Add(b.window)) {
		b.start = b.clock.Now()
		b.failures = 0
	}
}

type Event struct {
	Name    string
	Attempt int
	Error   error
}

type Hook func(Event)

type Policy struct {
	Retry         *RetryPolicy
	Timeout       *TimeoutPolicy
	Bulkhead      *Bulkhead
	RateLimiter   *RateLimiter
	Breaker       *CircuitBreaker
	FailureBudget *FailureBudget
	Hooks         []Hook
}

func (p Policy) Execute(ctx context.Context, name string, op Operation) error {
	if ctx == nil {
		return validationError("Policy.Execute", "context is required", nil)
	}
	if op == nil {
		return validationError("Policy.Execute", "operation is required", nil)
	}
	wrapped := op
	if p.FailureBudget != nil {
		inner := wrapped
		wrapped = func(ctx context.Context) error {
			if !p.FailureBudget.Allow() {
				return NewError(ErrorKindUnavailable, name, "failure budget exhausted", true)
			}
			err := inner(ctx)
			p.FailureBudget.Record(err)
			return err
		}
	}
	if p.Breaker != nil {
		inner := wrapped
		wrapped = func(ctx context.Context) error { return p.Breaker.Execute(ctx, inner) }
	}
	if p.RateLimiter != nil {
		inner := wrapped
		wrapped = func(ctx context.Context) error {
			if err := p.RateLimiter.Acquire(ctx); err != nil {
				return err
			}
			return inner(ctx)
		}
	}
	if p.Bulkhead != nil {
		inner := wrapped
		wrapped = func(ctx context.Context) error { return p.Bulkhead.Execute(ctx, inner) }
	}
	if p.Timeout != nil {
		inner := wrapped
		wrapped = func(ctx context.Context) error { return p.Timeout.Execute(ctx, inner) }
	}
	if p.Retry != nil {
		inner := wrapped
		wrapped = func(ctx context.Context) error { return p.Retry.Execute(ctx, inner) }
	}
	err := wrapped(ctx)
	p.emit(Event{Name: name, Error: err})
	return err
}

func (p Policy) emit(event Event) {
	for _, hook := range p.Hooks {
		if hook == nil {
			continue
		}
		func() {
			defer func() { _ = recover() }()
			hook(event)
		}()
	}
}
