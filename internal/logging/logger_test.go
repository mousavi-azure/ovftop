package logging

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLogAndTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	l.Info("connect", "connecting to %s", "esxi01")
	l.Warn("deploy", "retrying after transient error")
	l.Error("connect", "failed: %v", "timeout")

	lines, err := l.Tail(10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "[INFO] [connect] connecting to esxi01") {
		t.Errorf("unexpected line[0]: %q", lines[0])
	}
	if !strings.Contains(lines[2], "[ERROR] [connect] failed: timeout") {
		t.Errorf("unexpected line[2]: %q", lines[2])
	}
}

func TestTailTruncatesToN(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	for i := 0; i < 20; i++ {
		l.Info("test", "line %d", i)
	}

	lines, err := l.Tail(5)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[len(lines)-1], "line 19") {
		t.Errorf("expected last line to be line 19, got %q", lines[len(lines)-1])
	}
}

func TestNilLoggerIsNoOp(t *testing.T) {
	var l *Logger
	l.Info("x", "should not panic")
	if _, err := l.Tail(10); err != nil {
		t.Errorf("expected nil error from nil logger Tail, got %v", err)
	}
	if err := l.Close(); err != nil {
		t.Errorf("expected nil error from nil logger Close, got %v", err)
	}
}
