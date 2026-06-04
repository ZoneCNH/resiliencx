package testkit

import (
	"time"

	"github.com/ZoneCNH/resiliencx/pkg/resiliencx"
)

func Config(name string) resiliencx.Config {
	return resiliencx.Config{
		Name:    name,
		Timeout: time.Second,
	}
}
