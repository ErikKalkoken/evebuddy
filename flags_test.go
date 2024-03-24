package main

import (
	"log/slog"
	"testing"
)

func TestLogLevelFlagString(t *testing.T) {
	l := logLevelFlag{value: slog.LevelError}
	got := l.String()
	want := "ERROR"
	if got != want {
		t.Errorf("got=%v, want=%v", got, want)
	}
}

func TestLogLevelFlagSet1(t *testing.T) {
	var cases = []struct {
		in  string
		out slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
	}
	for _, c := range cases {
		l := logLevelFlag{}
		r := l.Set(c.in)
		if r != nil {
			t.Errorf("Set failed: %v", r)
		}
		if l.value != c.out {
			t.Errorf("Invalid level for \"%v\": got=%v, want=%v", c.in, l.value, c.out)
		}
	}
}

func TestLogLevelFlagSet2(t *testing.T) {
	l := logLevelFlag{}
	v := "xxx"
	r := l.Set(v)
	if r == nil {
		t.Errorf("Set should fail for: %v", v)
	}

}
