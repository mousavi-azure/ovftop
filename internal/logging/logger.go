// Package logging provides a simple append-only file logger for the
// application's own activity (connections, deployments, errors), viewable
// from the TUI's Logs screen (F8).
package logging

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

// Level is a log line's severity.
type Level string

const (
	Info  Level = "INFO"
	Warn  Level = "WARN"
	Error Level = "ERROR"
)

// Logger appends timestamped, leveled lines to a log file. A nil *Logger
// is safe to call methods on (they become no-ops), so callers that don't
// have one on hand (e.g. in tests) don't need to guard every call site.
type Logger struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

// Open opens (creating if needed) the log file at path for appending.
func Open(path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	return &Logger{path: path, f: f}, nil
}

// Close closes the underlying log file.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.f.Close()
}

// Log appends one leveled, categorized line.
func (l *Logger) Log(level Level, category, message string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	line := fmt.Sprintf("%s [%s] [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), level, category, message)
	_, _ = l.f.WriteString(line)
}

// Info, Warn, and Error are printf-style convenience wrappers around Log.
func (l *Logger) Info(category, format string, args ...any) {
	l.Log(Info, category, fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(category, format string, args ...any) {
	l.Log(Warn, category, fmt.Sprintf(format, args...))
}

func (l *Logger) Error(category, format string, args ...any) {
	l.Log(Error, category, fmt.Sprintf(format, args...))
}

// Tail returns up to the last n lines of the log file, oldest first.
func (l *Logger) Tail(n int) ([]string, error) {
	if l == nil {
		return nil, nil
	}
	l.mu.Lock()
	path := l.path
	l.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines, scanner.Err()
}
