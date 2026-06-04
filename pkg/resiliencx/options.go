package resiliencx

import "time"

type Option func(*options)

type clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

type clockFunc func() time.Time

func (fn clockFunc) Now() time.Time {
	return fn()
}

type options struct {
	metrics Metrics
	clock   clock
}

func defaultOptions() options {
	return options{
		metrics: NoopMetrics{},
		clock:   systemClock{},
	}
}

func WithMetrics(metrics Metrics) Option {
	return func(o *options) {
		if metrics != nil {
			o.metrics = metrics
		}
	}
}

func withClock(clock clock) Option {
	return func(o *options) {
		if clock != nil {
			o.clock = clock
		}
	}
}
