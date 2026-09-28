package xgoesi

import (
	"context"
	"errors"
	"log/slog"
)

// CanceledDowngradeLogger wraps a slog.Logger for use as a retryablehttp.LeveledLogger.
// It downgrades log lines caused by context cancellation (e.g. when the user stops
// an update scheduler mid-request) to Debug level, since these are expected and not
// actionable.
type CanceledDowngradeLogger struct {
	*slog.Logger
}

func (l CanceledDowngradeLogger) Error(msg string, keysAndValues ...any) {
	if isCanceledError(keysAndValues) {
		l.Logger.Debug(msg, keysAndValues...)
		return
	}
	l.Logger.Error(msg, keysAndValues...)
}

func (l CanceledDowngradeLogger) Warn(msg string, keysAndValues ...any) {
	if isCanceledError(keysAndValues) {
		l.Logger.Debug(msg, keysAndValues...)
		return
	}
	l.Logger.Warn(msg, keysAndValues...)
}

func (l CanceledDowngradeLogger) Info(msg string, keysAndValues ...any) {
	if isCanceledError(keysAndValues) {
		l.Logger.Debug(msg, keysAndValues...)
		return
	}
	l.Logger.Info(msg, keysAndValues...)
}

func isCanceledError(keysAndValues []any) bool {
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if keysAndValues[i] != "error" {
			continue
		}
		if err, ok := keysAndValues[i+1].(error); ok && errors.Is(err, context.Canceled) {
			return true
		}
	}
	return false
}
