package main

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestAppDirsDeleteAll(t *testing.T) {
	// given
	ap := appDirs{
		cache:    t.TempDir(),
		data:     t.TempDir(),
		log:      t.TempDir(),
		settings: t.TempDir(),
	}
	paths := []string{ap.cache, ap.data, ap.log, ap.settings}
	for _, p := range paths {
		x := filepath.Join(p, "dummy.txt")
		if err := os.WriteFile(x, []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range paths {
		assert.True(t, fileExists(p))
	}
	// when
	ap.deleteAll()
	// then
	for _, p := range paths {
		assert.False(t, fileExists(p))
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	panic(err)
}
