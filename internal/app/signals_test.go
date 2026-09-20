package app_test

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSignals_UniqueKey(t *testing.T) {
	t.Run("should return keys matching the expected format", func(t *testing.T) {
		s := app.NewSignals()
		got := s.UniqueKey()
		assert.Regexp(t, regexp.MustCompile(`^key-\d+$`), got)
	})
	t.Run("should return a different key on every call", func(t *testing.T) {
		s := app.NewSignals()
		k1 := s.UniqueKey()
		k2 := s.UniqueKey()
		assert.NotEqual(t, k1, k2)
	})
	t.Run("should return unique keys under concurrent use", func(t *testing.T) {
		s := app.NewSignals()
		const n = 500
		keys := make([]string, n)
		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				keys[i] = s.UniqueKey()
			}(i)
		}
		wg.Wait()

		seen := set.Of(keys...)
		xassert.Equal(t, n, seen.Size())
	})
}

func TestSignals_Shutdown(t *testing.T) {
	t.Run("should deliver Emit to listeners before shutdown began", func(t *testing.T) {
		s := app.NewSignals()
		received := make(chan struct{}, 1)
		s.UpdateStarted.AddListener(func(ctx context.Context, _ string) {
			received <- struct{}{}
		})
		s.UpdateStarted.Emit(context.Background(), "x")
		select {
		case <-received:
		case <-time.After(time.Second):
			t.Fatal("listener was not called")
		}
	})
	t.Run("should suppress Emit on a guarded signal after BeginShutdown", func(t *testing.T) {
		s := app.NewSignals()
		received := make(chan struct{}, 1)
		s.UpdateStarted.AddListener(func(ctx context.Context, _ string) {
			received <- struct{}{}
		})
		s.BeginShutdown()
		s.UpdateStarted.Emit(context.Background(), "x")
		select {
		case <-received:
			t.Fatal("listener should not have been called after shutdown began")
		case <-time.After(100 * time.Millisecond):
		}
	})
	t.Run("should still deliver AppShutdown after BeginShutdown", func(t *testing.T) {
		s := app.NewSignals()
		received := make(chan struct{}, 1)
		s.AppShutdown.AddListener(func(ctx context.Context, _ struct{}) {
			received <- struct{}{}
		})
		s.BeginShutdown()
		s.AppShutdown.Emit(context.Background(), struct{}{})
		select {
		case <-received:
		case <-time.After(time.Second):
			t.Fatal("AppShutdown listener was not called")
		}
	})
	t.Run("should report IsShuttingDown before and after BeginShutdown", func(t *testing.T) {
		s := app.NewSignals()
		assert.False(t, s.IsShuttingDown())
		s.BeginShutdown()
		assert.True(t, s.IsShuttingDown())
	})
	t.Run("should be safe under concurrent BeginShutdown and Emit calls", func(t *testing.T) {
		s := app.NewSignals()
		s.UpdateStarted.AddListener(func(ctx context.Context, _ string) {})
		const n = 100
		var wg sync.WaitGroup
		for range n {
			wg.Add(2)
			go func() {
				defer wg.Done()
				s.BeginShutdown()
			}()
			go func() {
				defer wg.Done()
				s.UpdateStarted.Emit(context.Background(), "x")
			}()
		}
		wg.Wait()
	})
}

func TestSignals_PseudoUniqueID(t *testing.T) {
	t.Run("should return an ID matching the expected format", func(t *testing.T) {
		s := app.NewSignals()
		got := s.PseudoUniqueID()
		assert.Regexp(t, regexp.MustCompile(`^\d+-\d+$`), got)
	})
	t.Run("should return unique IDs across many calls", func(t *testing.T) {
		s := app.NewSignals()
		const n = 500
		ids := make([]string, n)
		for i := range n {
			ids[i] = s.PseudoUniqueID()
		}
		seen := set.Of(ids...)
		xassert.Equal(t, n, seen.Size())
	})
}
