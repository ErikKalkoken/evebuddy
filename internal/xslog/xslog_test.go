package xslog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	h := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})
	return slog.New(NewCanceledDowngradingHandler(h))
}

func TestCanceledDowngradingHandler(t *testing.T) {
	t.Run("should downgrade a canceled error to debug when debug is enabled", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(&buf, slog.LevelDebug)
		err := fmt.Errorf("update section: %w", context.Canceled)
		logger.Error("update failed", "error", err)
		out := buf.String()
		assert.Contains(t, out, "level=DEBUG")
		assert.Contains(t, out, "update failed")
		assert.NotContains(t, out, "level=ERROR")
	})
	t.Run("should drop a canceled error entirely when debug is not enabled", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(&buf, slog.LevelInfo)
		err := fmt.Errorf("update section: %w", context.Canceled)
		logger.Error("update failed", "error", err)
		assert.Empty(t, buf.String())
	})
	t.Run("should leave a genuine error unchanged", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(&buf, slog.LevelInfo)
		err := errors.New("boom")
		logger.Error("update failed", "error", err)
		out := buf.String()
		assert.Contains(t, out, "level=ERROR")
		assert.Contains(t, out, "boom")
	})
	t.Run("should leave a non-error record unchanged", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(&buf, slog.LevelInfo)
		logger.Info("all good")
		out := buf.String()
		assert.Contains(t, out, "level=INFO")
		assert.Contains(t, out, "all good")
	})
	t.Run("should support WithAttrs and WithGroup", func(t *testing.T) {
		var buf bytes.Buffer
		logger := newTestLogger(&buf, slog.LevelDebug)
		logger = logger.With("component", "test").WithGroup("g")
		err := fmt.Errorf("update section: %w", context.Canceled)
		logger.Error("update failed", "error", err)
		out := buf.String()
		require.NotEmpty(t, out)
		assert.Contains(t, out, "component=test")
		assert.Contains(t, out, "level=DEBUG")
	})
}
