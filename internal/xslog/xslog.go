// Package xslog extends the standard log/slog package.
package xslog

import (
	"context"
	"errors"
	"log/slog"
)

// NewCanceledDowngradingHandler wraps h so that any log record at or above
// [slog.LevelError] whose logged error wraps [context.Canceled] is downgraded
// to [slog.LevelDebug] before being passed on to h.
//
// A canceled context is expected during graceful shutdown (e.g. when a
// background update ticker is stopped mid-flight) and elsewhere in the app
// where an in-flight operation is intentionally aborted; it is not a genuine
// failure and should not be logged as one.
func NewCanceledDowngradingHandler(h slog.Handler) slog.Handler {
	return &canceledDowngradingHandler{Handler: h}
}

type canceledDowngradingHandler struct {
	slog.Handler
}

func (h *canceledDowngradingHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError && recordHasCanceledError(r) {
		if !h.Handler.Enabled(ctx, slog.LevelDebug) {
			return nil
		}
		nr := slog.NewRecord(r.Time, slog.LevelDebug, r.Message, r.PC)
		r.Attrs(func(a slog.Attr) bool {
			nr.AddAttrs(a)
			return true
		})
		r = nr
	}
	return h.Handler.Handle(ctx, r)
}

func (h *canceledDowngradingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &canceledDowngradingHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *canceledDowngradingHandler) WithGroup(name string) slog.Handler {
	return &canceledDowngradingHandler{Handler: h.Handler.WithGroup(name)}
}

func recordHasCanceledError(r slog.Record) bool {
	found := false
	r.Attrs(func(a slog.Attr) bool {
		if err, ok := a.Value.Any().(error); ok && errors.Is(err, context.Canceled) {
			found = true
			return false
		}
		return true
	})
	return found
}
