package testutil_test

import (
	"bytes"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestNewDBInMemory_LogLevel(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	t.Run("should suppress migration logs by default", func(t *testing.T) {
		// given
		logBuf.Reset()
		// when
		db, _, _ := testutil.NewDBInMemory()
		defer db.Close()
		// then
		assert.NotContains(t, logBuf.String(), "migration")
	})
	t.Run("should show migration logs when a lower log level is requested", func(t *testing.T) {
		// given
		logBuf.Reset()
		// when
		db, _, _ := testutil.NewDBInMemory(testutil.WithLogLevel(slog.LevelInfo))
		defer db.Close()
		// then
		assert.Contains(t, logBuf.String(), "Successfully applied new migration")
	})
	t.Run("should restore the previous log level afterward", func(t *testing.T) {
		// given
		prev := slog.SetLogLoggerLevel(slog.LevelDebug)
		defer slog.SetLogLoggerLevel(prev)
		// when
		db, _, _ := testutil.NewDBInMemory()
		defer db.Close()
		// then
		got := slog.SetLogLoggerLevel(slog.LevelDebug)
		assert.Equal(t, slog.LevelDebug, got)
	})
}

func TestNewDBOnDisk_LogLevel(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	t.Run("should suppress migration logs by default", func(t *testing.T) {
		// given
		logBuf.Reset()
		// when
		db, _, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		// then
		assert.NotContains(t, logBuf.String(), "migration")
	})
	t.Run("should show migration logs when a lower log level is requested", func(t *testing.T) {
		// given
		logBuf.Reset()
		// when
		db, _, _ := testutil.NewDBOnDisk(t, testutil.WithLogLevel(slog.LevelInfo))
		defer db.Close()
		// then
		assert.Contains(t, logBuf.String(), "Successfully applied new migration")
	})
}
