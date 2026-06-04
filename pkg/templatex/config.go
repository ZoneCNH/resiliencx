package templatex

import (
	"errors"
	"time"

	"github.com/ZoneCNH/xlib-standard/internal/sanitize"
	"github.com/ZoneCNH/xlib-standard/internal/validation"
)

type Config struct {
	Name    string
	Timeout time.Duration
	Secret  string
}

// SanitizedConfig is a logging-safe copy of Config where the Secret field
// has been masked. It is produced by Config.Sanitize and is safe to emit in
// log lines, metrics labels, or error messages without leaking credentials.
type SanitizedConfig struct {
	Name    string
	Timeout time.Duration
	Secret  string
}

// Validate checks that the Config fields satisfy their constraints.
// It returns a validation error if Name is empty or Timeout is negative,
// and nil otherwise. Validate should be called before passing Config to New.
func (c Config) Validate() error {
	if err := validation.RequireNonEmpty("name", c.Name); err != nil {
		return validationError("Config.Validate", err.Error(), err)
	}
	if c.Timeout < 0 {
		err := errors.New("timeout must not be negative")
		return validationError("Config.Validate", err.Error(), err)
	}
	return nil
}

// Sanitize returns a SanitizedConfig derived from c with the Secret field
// masked for safe logging. All other fields are copied as-is.
func (c Config) Sanitize() SanitizedConfig {
	return SanitizedConfig{
		Name:    c.Name,
		Timeout: c.Timeout,
		Secret:  sanitize.Secret(c.Secret),
	}
}
