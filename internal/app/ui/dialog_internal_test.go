package ui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
)

// jobRunnerFake runs jobs synchronously with ctx, or refuses them.
type jobRunnerFake struct {
	ctx    context.Context
	refuse bool
}

func (r jobRunnerFake) Go(job func(ctx context.Context)) bool {
	if r.refuse {
		return false
	}
	job(r.ctx)
	return true
}

func TestLoadingConfirm(t *testing.T) {
	test.NewTempApp(t)
	confirm := func(r JobRunner) (called, done bool) {
		_, b := newLoadingConfirm(r, "title", "message", "OK", widget.DangerImportance, func(ctx context.Context) {
			called = true
		}, test.NewWindow(nil))
		b.OnAction(func() { done = true })
		return called, done
	}

	t.Run("runs callback and reports done", func(t *testing.T) {
		called, done := confirm(jobRunnerFake{ctx: context.Background()})
		assert.True(t, called)
		assert.True(t, done)
	})
	t.Run("reports done without callback when runner refuses", func(t *testing.T) {
		called, done := confirm(jobRunnerFake{refuse: true})
		assert.False(t, called)
		assert.True(t, done)
	})
	t.Run("skips done when canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called, done := confirm(jobRunnerFake{ctx: ctx})
		assert.True(t, called)
		assert.False(t, done)
	})
}
