package main

import (
	"fmt"
	"time"

	"github.com/ZoneCNH/resiliencx/pkg/resiliencx"
)

func main() {
	cfg := resiliencx.Config{
		Name:    "resiliencx",
		Timeout: time.Second,
		Secret:  "example",
	}

	fmt.Println(cfg.Sanitize().Secret)
}
