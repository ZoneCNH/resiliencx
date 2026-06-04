package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func runMainGuard(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli main-guard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	context := flags.String("context", envDefault("XLIB_CONTEXT", "local_write"), "execution context")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !validContext(*context) {
		write(stderr, "ERROR: invalid context %q\n", *context)
		return 2
	}
	branch := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if *context == "local_write" && (branch == "main" || branch == "master") {
		return emitReport(stdout, "main-guard", "failed", nil, []string{"local_write is forbidden on " + branch})
	}
	return emitReport(stdout, "main-guard", "passed", []string{"context=" + *context, "branch=" + fallback(branch, "unknown")}, nil)
}

func runWorktreeGuard(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWorktreeGate("worktree-guard", args, stdout, stderr)
}

func runWorktreeCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWorktreeGate("worktree-check", args, stdout, stderr)
}

func runWorktreeGate(command string, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	context := flags.String("context", envDefault("XLIB_CONTEXT", "local_write"), "execution context")
	flags.Bool("json", false, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		write(stderr, "ERROR: %s invalid arguments: unexpected positional argument %q\n", command, flags.Arg(0))
		return 2
	}
	if !validContext(*context) {
		write(stderr, "ERROR: invalid context %q\n", *context)
		return 2
	}
	details, gaps := evaluateWorktreeGate(*context)
	if len(gaps) > 0 {
		return emitReport(stdout, command, "failed", details, gaps)
	}
	return emitReport(stdout, command, "passed", details, nil)
}

func evaluateWorktreeGate(context string) ([]string, []string) {
	top := gitOutput("rev-parse", "--show-toplevel")
	common := gitOutput("rev-parse", "--path-format=absolute", "--git-common-dir")
	isWorkerTree := strings.Contains(top, string(filepath.Separator)+".worktree"+string(filepath.Separator)) || strings.Contains(top, string(filepath.Separator)+".worktrees"+string(filepath.Separator)) || strings.Contains(common, string(filepath.Separator)+"worktrees"+string(filepath.Separator))
	details := []string{"context=" + context, "top=" + fallback(top, "unknown")}
	if context == "local_write" && !isWorkerTree {
		return details, []string{"local_write requires a worker worktree"}
	}
	return details, nil
}

func runContextCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("context-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("context-check", err, stderr)
	}
	required := []string{"docs/goal", "docs/goal/goal.md"}
	var gaps []string
	for _, path := range required {
		if !fileExists(path) {
			gaps = append(gaps, "missing "+path)
		}
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: context-check found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "context-check", "failed", nil, gaps)
	}
	return emitReport(stdout, "context-check", "passed", []string{"docs/goal context is present"}, nil)
}

func runSpecCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("spec-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("spec-check", err, stderr)
	}
	if !fileExists("docs") {
		write(stderr, "ERROR: spec-check found 1 gap(s)\n")
		return emitReport(stdout, "spec-check", "failed", nil, []string{"missing docs"})
	}
	found := false
	var gaps []string
	paths, err := trackedDocsMarkdownFiles()
	if err != nil {
		gaps = append(gaps, "scan docs: "+err.Error())
	}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			gaps = append(gaps, "read "+path+": "+readErr.Error())
			continue
		}
		if strings.Contains(string(data), "REQ-") {
			found = true
		}
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: spec-check found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "spec-check", "failed", nil, gaps)
	}
	details := []string{fmt.Sprintf("scanned_markdown=%d", len(paths))}
	if !found {
		details = append(details, "warning: no docs markdown file contains REQ-")
	}
	return emitReport(stdout, "spec-check", "passed", details, nil)
}

func trackedDocsMarkdownFiles() ([]string, error) {
	out, err := exec.Command("git", "ls-files", "-z", "--", "docs").Output()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, path := range strings.Split(string(out), "\x00") {
		if path == "" || filepath.Ext(path) != ".md" {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func runDesignCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("design-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("design-check", err, stderr)
	}
	if !fileExists("docs/adr") {
		return emitReport(stdout, "design-check", "passed", []string{"warning: optional docs/adr not present"}, nil)
	}
	return emitReport(stdout, "design-check", "passed", []string{"docs/adr is present"}, nil)
}

func runTaskCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("task-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("task-check", err, stderr)
	}
	switch {
	case fileExists(".agent/command-registry.yaml"):
		return emitReport(stdout, "task-check", "passed", []string{".agent/command-registry.yaml is present"}, nil)
	case fileExists(".agent/registry/commands.yaml"):
		return emitReport(stdout, "task-check", "passed", []string{"legacy .agent/registry/commands.yaml is present"}, nil)
	default:
		return emitReport(stdout, "task-check", "passed", []string{"warning: command registry not present"}, nil)
	}
}

func runPRCheck(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli pr-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	context := flags.String("context", envDefault("XLIB_CONTEXT", "local_write"), "execution context")
	dryRun := flags.Bool("dry-run", false, "")
	flags.Bool("json", false, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		write(stderr, "ERROR: pr-check invalid arguments: unexpected positional argument %q\n", flags.Arg(0))
		return 2
	}
	if !validContext(*context) {
		write(stderr, "ERROR: invalid context %q\n", *context)
		return 2
	}
	if *dryRun {
		return emitReport(stdout, "pr-check", "passed", []string{"mode=dry-run", "context=" + *context, "delegates=worktree-check,lint,test"}, nil)
	}
	if details, gaps := evaluateWorktreeGate(*context); len(gaps) > 0 {
		return emitReport(stdout, "pr-check", "failed", details, gaps)
	}
	if code := runExternal(stdin, stderr, stderr, "make", "lint"); code != 0 {
		return emitReport(stdout, "pr-check", "failed", nil, []string{fmt.Sprintf("make lint exited %d", code)})
	}
	if code := runExternal(stdin, stderr, stderr, "make", "test"); code != 0 {
		return emitReport(stdout, "pr-check", "failed", nil, []string{fmt.Sprintf("make test exited %d", code)})
	}
	return emitReport(stdout, "pr-check", "passed", []string{"context=" + *context, "make lint passed", "make test passed"}, nil)
}
