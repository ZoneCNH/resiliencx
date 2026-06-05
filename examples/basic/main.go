package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ZoneCNH/resiliencx/pkg/resiliencx"
)

func main() {
	run(os.Stdout, os.Stderr, resiliencx.Config{Name: "resiliencx"})
}

func run(stdout, stderr io.Writer, cfg resiliencx.Config) {
	client, err := resiliencx.New(context.Background(), cfg)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "create client: %v\n", err)
		return
	}
	defer func() {
		_ = client.Close(context.Background())
	}()

	_, _ = fmt.Fprintln(stdout, resiliencx.ModuleName)
}
