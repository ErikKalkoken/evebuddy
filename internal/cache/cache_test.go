package cache_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/cache"
)

func TestCacheExported(t *testing.T) {
	t.Parallel()
	c := cache.New()
	t.Run("can set a key", func(t *testing.T) {
		// when
		c.Set("k1", "xxx", time.Second*100)
		// then
		assert.True(t, c.Exists("k1"))
	})
	t.Run("can get a key", func(t *testing.T) {
		// given
		c.Set("k2", "xxx", time.Second*100)
		// when
		o, ok := c.Get("k2")
		// then
		if assert.True(t, ok) {
			assert.Equal(t, "xxx", o.(string))
		}
	})
	t.Run("can check if a key exists", func(t *testing.T) {
		// given
		c.Set("k6", "xxx", time.Second*100)
		// when/then
		assert.True(t, c.Exists("k6"))
		assert.False(t, c.Exists("other"))
	})
	t.Run("can set key that never expires", func(t *testing.T) {
		// given
		c.Set("k7", "xxx", 0)
		// when/then
		assert.True(t, c.Exists("k7"))
	})
	t.Run("should report when key is expired", func(t *testing.T) {
		// given
		c.Set("k3", "xxx", time.Millisecond*10)
		// when
		time.Sleep(time.Millisecond * 50)
		o, ok := c.Get("k3")
		// then
		assert.False(t, ok)
		assert.Nil(t, o)
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		c.Set("k4", "xxx", time.Second*100)
		// when
		c.Delete("k4")
		// then
		assert.False(t, c.Exists("k4"))
	})
	t.Run("can clear all keys", func(t *testing.T) {
		// given
		c2 := cache.New()
		c2.Set("dummy-1", "xxx", time.Second*100)
		c2.Set("dummy-2", "xxx", time.Second*100)
		// when
		c2.Clear()
		// then
		assert.False(t, c2.Exists("dummy-1"))
		assert.False(t, c2.Exists("dummy-2"))
	})
	t.Run("can remove all expired keys", func(t *testing.T) {
		// given
		c3 := cache.New()
		c3.Set("dummy-1", "xxx", time.Millisecond*20)
		c3.Set("dummy-2", "xxx", time.Second*100)
		time.Sleep(time.Millisecond * 50)
		// when
		c3.Reorganize()
		// then
		assert.False(t, c3.Exists("dummy-1"))
		assert.True(t, c3.Exists("dummy-2"))
	})
}
