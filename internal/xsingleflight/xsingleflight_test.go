package xsingleflight_test

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"

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

	t.Run("should report shared=true when calls are merged", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var g singleflight.Group
			block := make(chan struct{})

			var (
				shared1, shared2 bool
				wg               sync.WaitGroup
			)

			wg.Add(2)

			// 1. First call registers and blocks inside fn()
			go func() {
				defer wg.Done()
				_, _, shared1 = xsingleflight.Do(&g, "key", func() (int, error) {
					<-block
					return 100, nil
				})
			}()

			// Wait until goroutine 1 is durably blocked waiting on `<-block`
			synctest.Wait()

			// 2. Second call attempts to execute under the same key and joins goroutine 1
			go func() {
				defer wg.Done()
				_, _, shared2 = xsingleflight.Do(&g, "key", func() (int, error) {
					return 100, nil
				})
			}()

			// Wait until goroutine 2 joins the singleflight execution and blocks
			synctest.Wait()

			// Unblock the execution and let both finish
			close(block)
			synctest.Wait()
			wg.Wait()

			assert.True(t, shared1 || shared2, "At least one call should be marked as shared")
		})
	})
}

func TestDo_TypeMismatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := &singleflight.Group{}
		key := "shared-key"
		block := make(chan struct{})

		var (
			val int
			err error
		)

		// 1. Launch the first call expecting a STRING and block inside its execution
		go func() {
			xsingleflight.Do(g, key, func() (string, error) {
				<-block // Hold execution so subsequent calls join this group key
				return "i am a string", nil
			})
		}()

		// Wait until the goroutine above is blocked waiting on the channel
		synctest.Wait()

		// 2. Launch the second call using the same key expecting an INT
		go func() {
			val, err, _ = xsingleflight.Do(g, key, func() (int, error) {
				return 42, nil
			})
		}()

		// Wait until the second goroutine joins and blocks on the singleflight Group
		synctest.Wait()

		// 3. Unblock the original execution and wait for all goroutines in the bubble to complete
		close(block)
		synctest.Wait()

		// Assertions using testify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type conversion failed")
		assert.Contains(t, err.Error(), "got string, want int")
		assert.Equal(t, 0, val, "Should return zero value on type conversion failure")
	})
}
