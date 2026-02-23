package xsingleflight_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

func TestDo(t *testing.T) {
	t.Run("should return value when function suceeds", func(t *testing.T) {
		var g singleflight.Group
		v, err, shared := xsingleflight.Do(&g, "dummy", func() (int, error) {
			return 42, nil
		})
		require.NoError(t, err)
		xassert.Equal(t, 42, v)
		assert.False(t, shared)
	})
	t.Run("should return error when function fails", func(t *testing.T) {
		var g singleflight.Group
		_, err, shared := xsingleflight.Do(&g, "dummy", func() (int, error) {
			return 0, fmt.Errorf("some error")
		})
		require.Error(t, err)
		assert.False(t, shared)
	})
	t.Run("should return error when no group passed", func(t *testing.T) {
		_, err, shared := xsingleflight.Do(nil, "dummy", func() (int, error) {
			return 0, fmt.Errorf("some error")
		})
		require.Error(t, err)
		assert.False(t, shared)
	})
	t.Run("Error - Type Mismatch", func(t *testing.T) {
		g := &singleflight.Group{}
		key := "shared-key"
		block := make(chan struct{})
		finished := make(chan struct{})

		// 1. Start a call that returns a STRING and blocks
		go func() {
			xsingleflight.Do(g, key, func() (string, error) {
				<-block // Hold the execution so the second call joins this one
				return "i am a string", nil
			})
			close(finished)
		}()

		// Give the goroutine a moment to register with the group
		// In a real test, you'd use a more robust sync mechanism
		time.Sleep(10 * time.Millisecond)

		// 2. Try to call the same key expecting an INT
		// This will join the first call and eventually receive a string
		close(block)
		val, err, _ := xsingleflight.Do(g, key, func() (int, error) {
			return 42, nil
		})

		<-finished

		// Assertions
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type conversion failed")
		assert.Equal(t, 0, val, "Should return zero value on failure")
		assert.Contains(t, err.Error(), "got string, want int")
	})
}
