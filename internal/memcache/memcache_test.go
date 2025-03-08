package memcache_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

func TestMemcache(t *testing.T) {
	t.Parallel()
	c := memcache.New()
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

}

func TestMemcache2(t *testing.T) {
	t.Parallel()
	t.Run("can clear all keys", func(t *testing.T) {
		// given
		c := memcache.New()
		c.Set("dummy-1", "xxx", time.Second*100)
		c.Set("dummy-2", "xxx", time.Second*100)
		// when
		c.Clear()
		// then
		assert.False(t, c.Exists("dummy-1"))
		assert.False(t, c.Exists("dummy-2"))
	})
	t.Run("can close cache", func(t *testing.T) {
		c := memcache.New()
		c.Close()
	})
	t.Run("can start cache without automatic clean-up", func(t *testing.T) {
		c := memcache.NewWithTimeout(0)
		c.Close()
	})
	t.Run("can run clean-up", func(t *testing.T) {
		c := memcache.NewWithTimeout(0)
		c.CleanUp()
		c.Close()
	})
}
