package templatex

import (
	"context"
	"errors"
)

type ErrorKind string

// Predefined error kinds used throughout the package.
const (
	// ErrorKindConfig indicates a configuration error (e.g. missing or invalid fields).
	ErrorKindConfig ErrorKind = "config"
	// ErrorKindValidation indicates a user-input validation failure.
	ErrorKindValidation ErrorKind = "validation"
	// ErrorKindConnection indicates a network connection failure.
	ErrorKindConnection ErrorKind = "connection"
	// ErrorKindUnavailable indicates the remote service is temporarily unavailable.
	ErrorKindUnavailable ErrorKind = "unavailable"
	// ErrorKindTimeout indicates an operation exceeded its deadline.
	ErrorKindTimeout ErrorKind = "timeout"
	// ErrorKindAuth indicates an authentication or authorization failure.
	ErrorKindAuth ErrorKind = "auth"
	// ErrorKindConflict indicates a conflict with the current resource state (e.g. optimistic locking).
	ErrorKindConflict ErrorKind = "conflict"
	// ErrorKindRateLimit indicates the operation was rejected by a rate limiter.
	ErrorKindRateLimit ErrorKind = "rate_limit"
	// ErrorKindInternal indicates an unexpected internal error.
	ErrorKindInternal ErrorKind = "internal"
)

// Error is the structured error type returned by all package operations.
// It carries a Kind for programmatic classification, an Op describing the
// operation that failed, a human-readable Message, an optional wrapped Cause,
// and a Retryable flag indicating whether the caller may safely retry.
type Error struct {
	Kind      ErrorKind
	Op        string
	Message   string
	Cause     error
	Retryable bool
}

// NewError creates a new *Error with the given kind, operation name, message,
// and retryable flag. Use WrapError instead when wrapping an existing error
// as the cause.
func NewError(kind ErrorKind, op string, message string, retryable bool) *Error {
	return newError(kind, op, message, retryable, nil)
}

// WrapError creates a new *Error that wraps an existing cause.
// If message is empty, the cause's message is used. The resulting error
// supports errors.Is and errors.As via its Unwrap method.
func WrapError(kind ErrorKind, op string, message string, retryable bool, cause error) *Error {
	return newError(kind, op, message, retryable, cause)
}

// Error returns a human-readable string in the form "kind: op: message".
// Calling Error on a nil *Error returns an empty string.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	message := string(e.Kind)
	if e.Op != "" {
		message += ": " + e.Op
	}
	if e.Message != "" {
		message += ": " + e.Message
	}
	if e.Message == "" && e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

// Unwrap returns the underlying cause of the error, or nil if no cause was
// provided. It enables standard errors.Is and errors.As traversal.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// IsKind reports whether err (or any error in its chain) is an *Error whose
// Kind matches the given kind. It returns false if err is nil or not an *Error.
func IsKind(err error, kind ErrorKind) bool {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind == kind
	}
	return false
}

func newError(kind ErrorKind, op string, message string, retryable bool, cause error) *Error {
	if message == "" && cause != nil {
		message = cause.Error()
	}
	return &Error{
		Kind:      kind,
		Op:        op,
		Message:   message,
		Cause:     cause,
		Retryable: retryable,
	}
}

func validationError(op string, message string, cause error) *Error {
	return newError(ErrorKindValidation, op, message, false, cause)
}

func contextError(op string, cause error) *Error {
	kind := ErrorKindUnavailable
	retryable := false
	if errors.Is(cause, context.DeadlineExceeded) {
		kind = ErrorKindTimeout
		retryable = true
	}
	return newError(kind, op, "", retryable, cause)
}

func errorKind(err error) ErrorKind {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind
	}
	return ErrorKindInternal
}
