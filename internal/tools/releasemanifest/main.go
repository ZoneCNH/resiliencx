// SPDX-License-Identifier: Apache-2.0
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

var exit = os.Exit

// Release manifest CLI 的 gate 以 main.go 作为契约锚点。
const (
	defaultReleaseVersion       = "v0.4.14"
	defaultManifestOutputPath   = "release/manifest/latest.json"
	defaultManifestChecksumPath = "release/manifest/latest.json.sha256"
)

func main() {
	exit(runCLI(os.Args[0], os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(name string, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", defaultManifestOutputPath, "release manifest output path")
	verify := flags.String("verify", "", "verify an existing release manifest instead of generating one")
	requirePassed := flags.Bool("require-passed", false, "require all release checks to be passed during verification")
	requireClean := flags.Bool("require-clean", false, "require a clean git tree during verification")
	expectVersion := flags.String("expect-version", "", "require the manifest version to match this release version during verification")
	minScore := flags.Float64("min-score", 0, "require the release score to be at least this value during verification")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *verify != "" {
		if err := verifyManifest(*verify, *requirePassed, *requireClean, *expectVersion, *minScore); err != nil {
			return printCLIError(stderr, err)
		}
		return printCLIStatus(stdout, "release evidence verified: %s\n", *verify)
	}

	manifest, err := buildManifest()
	if err != nil {
		return printCLIError(stderr, err)
	}
	if err := writeManifest(*out, manifest); err != nil {
		return printCLIError(stderr, err)
	}
	return printCLIStatus(stdout, "generated %s\n", *out)
}

func printCLIError(w io.Writer, err error) int {
	return printCLIMessage(w, 1, "ERROR: %v\n", err)
}

func printCLIStatus(w io.Writer, format string, args ...any) int {
	return printCLIMessage(w, 0, format, args...)
}

func printCLIMessage(w io.Writer, exitCode int, format string, args ...any) int {
	_, err := fmt.Fprintf(w, format, args...)
	if err != nil {
		return 1
	}
	return exitCode
}
