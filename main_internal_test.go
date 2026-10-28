package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"github.com/juju/mutex/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

// Crash output must be reset before the temp dir is removed,
// because Windows can't delete open files.
func TestSetupCrashFile(t *testing.T) {
	t.Run("should create crash file", func(t *testing.T) {
		// given
		p := filepath.Join(t.TempDir(), crashFileName)
		t.Cleanup(func() {
			debug.SetCrashOutput(nil, debug.CrashOptions{})
		})
		// when
		err := setupCrashFile(p)
		// then
		require.NoError(t, err)
		_, err = os.Stat(p)
		assert.NoError(t, err)
	})

	t.Run("should keep content of existing crash file", func(t *testing.T) {
		// given
		p := filepath.Join(t.TempDir(), crashFileName)
		t.Cleanup(func() {
			debug.SetCrashOutput(nil, debug.CrashOptions{})
		})
		err := os.WriteFile(p, []byte("old crash"), 0644)
		require.NoError(t, err)
		// when
		err = setupCrashFile(p)
		// then
		require.NoError(t, err)
		data, err := os.ReadFile(p)
		require.NoError(t, err)
		xassert.Equal(t, "old crash", string(data))
	})

	t.Run("should return error when folder does not exist", func(t *testing.T) {
		// given
		p := filepath.Join(t.TempDir(), "missing", crashFileName)
		// when
		err := setupCrashFile(p)
		// then
		assert.Error(t, err)
	})
}

func TestRealtime(t *testing.T) {
	t.Run("should return current time", func(t *testing.T) {
		// given
		var r realtime
		// when
		got := r.Now()
		// then
		xassert.EqualDuration(t, 0, time.Since(got), time.Second)
	})

	t.Run("should send time after duration", func(t *testing.T) {
		// given
		var r realtime
		start := time.Now()
		// when
		c := r.After(10 * time.Millisecond)
		// then
		select {
		case got := <-c:
			assert.GreaterOrEqual(t, got.Sub(start), 10*time.Millisecond)
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	})
}

func TestEnsureSingleInstance(t *testing.T) {
	t.Run("should acquire mutex when no other instance running", func(t *testing.T) {
		// given
		name := newTestMutexName(t, "evebuddy-test-acquire")
		// when
		mu, err := ensureSingleInstance(name)
		// then
		require.NoError(t, err)
		mu.Release()
	})

	t.Run("should return timeout error when another instance running", func(t *testing.T) {
		// given
		name := newTestMutexName(t, "evebuddy-test-timeout")
		mu, err := ensureSingleInstance(name)
		require.NoError(t, err)
		defer mu.Release()
		// when
		_, err = ensureSingleInstance(name)
		// then
		assert.ErrorIs(t, err, mutex.ErrTimeout)
	})

	t.Run("should acquire mutex again after release", func(t *testing.T) {
		// given
		name := newTestMutexName(t, "evebuddy-test-release")
		mu, err := ensureSingleInstance(name)
		require.NoError(t, err)
		mu.Release()
		// when
		mu, err = ensureSingleInstance(name)
		// then
		require.NoError(t, err)
		mu.Release()
	})

	t.Run("should return error when name is invalid", func(t *testing.T) {
		// when
		_, err := ensureSingleInstance("invalid.name")
		// then
		assert.Error(t, err)
		assert.NotErrorIs(t, err, mutex.ErrTimeout)
	})
}

// newTestMutexName returns a mutex name unique to this test process
// and removes the lock file after the test.
func newTestMutexName(t *testing.T, base string) string {
	t.Helper()
	name := fmt.Sprintf("%s-%d", base, os.Getpid())
	t.Cleanup(func() {
		os.Remove(filepath.Join(os.TempDir(), "juju-"+name)) // lock file on Unix only
	})
	return name
}

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"upper case", "DEBUG", slog.LevelDebug},
		{"mixed case", "Warn", slog.LevelWarn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := parseLogLevel(tc.in)
			// then
			require.NoError(t, err)
			xassert.Equal(t, tc.want, got)
		})
	}

	t.Run("should return error for invalid level", func(t *testing.T) {
		// when
		_, err := parseLogLevel("verbose")
		// then
		require.Error(t, err)
		for _, s := range []string{"debug", "info", "warn", "error"} {
			assert.Contains(t, err.Error(), s)
		}
	})
}

func TestCalcCPULimit(t *testing.T) {
	cases := []struct {
		name  string
		total int
		want  int
	}{
		{"one CPU", 1, 1},
		{"two CPUs", 2, 1},
		{"odd number of CPUs", 7, 3},
		{"even number of CPUs", 8, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := calcCPULimit(tc.total)
			// then
			xassert.Equal(t, tc.want, got)
		})
	}
}

func TestNewAppPaths(t *testing.T) {
	// given
	dataDir := filepath.Join("data", "evebuddy")
	// when
	got := newAppPaths(dataDir)
	// then
	want := appPaths{
		db:        filepath.Join(dataDir, "evebuddy.sqlite"),
		logDir:    filepath.Join(dataDir, "log"),
		logFile:   filepath.Join(dataDir, "log", "evebuddy.log"),
		crashFile: filepath.Join(dataDir, "log", "crash.txt"),
	}
	xassert.Equal(t, want, got)
}
