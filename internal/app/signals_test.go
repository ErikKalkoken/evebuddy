package app_test

import (
	"regexp"
	"sync"
	"testing"

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
