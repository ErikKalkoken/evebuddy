package xsingleflight_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

func TestDo(t *testing.T) {
	var g singleflight.Group
	t.Run("should return value when function suceeds", func(t *testing.T) {
		v, err, shared := xsingleflight.Do(&g, "dummy", func() (int, error) {
			return 42, nil
		})
		require.NoError(t, err)
		xassert.Equal(t, 42, v)
		assert.False(t, shared)
	})
	t.Run("should return error when function fails", func(t *testing.T) {
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
}
