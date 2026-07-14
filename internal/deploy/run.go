package deploy

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// EventKind distinguishes a scrolling log line from the terminal outcome.
type EventKind int

const (
	EventLine EventKind = iota
	EventDone
)

// Event is one unit of output from a running ovftool invocation.
type Event struct {
	Kind    EventKind
	Text    string // valid when Kind == EventLine
	Percent int    // parsed progress percentage, or -1 if none was found
	Success bool   // valid when Kind == EventDone
	Err     error  // valid when Kind == EventDone and !Success
}

var percentRe = regexp.MustCompile(`(\d{1,3})\s*%`)

// Run executes ovftool with args, streaming output to onEvent as it
// arrives and finishing with exactly one EventDone. Output is split on
// both '\n' and '\r' — ovftool rewrites a single progress line in place
// using carriage returns, so treating '\r' as a line break is what turns
// that into a sequence of discrete percentage updates. Canceling ctx
// kills the process.
func Run(ctx context.Context, ovftoolPath string, args []string, onEvent func(Event)) error {
	cmd := exec.CommandContext(ctx, ovftoolPath, args...)

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		_ = r.Close()
		_ = w.Close()
		return err
	}
	_ = w.Close() // our copy; the child's fd keeps it alive until it exits

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	scanner.Split(splitOnLineOrCR)

	for scanner.Scan() {
		text := strings.TrimRight(scanner.Text(), " \t")
		if text == "" {
			continue
		}
		percent := -1
		if m := percentRe.FindStringSubmatch(text); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				percent = v
			}
		}
		onEvent(Event{Kind: EventLine, Text: text, Percent: percent})
	}
	_ = r.Close()

	waitErr := cmd.Wait()
	onEvent(Event{Kind: EventDone, Success: waitErr == nil, Err: waitErr})
	return waitErr
}

func splitOnLineOrCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
