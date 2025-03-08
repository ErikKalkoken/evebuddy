package memcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemcache(t *testing.T) {
	t.Parallel()
	t.Run("can remove all expired keys", func(t *testing.T) {
		// given
		c := New()
		c.Set("dummy-1", "xxx", time.Millisecond*20)
		c.Set("dummy-2", "xxx", time.Second*100)
		time.Sleep(time.Millisecond * 50)
		// when
		c.CleanUp()
		// then
		_, found := c.items.Load("dummy-1")
		assert.False(t, found)
		_, found = c.items.Load("dummy-2")
		assert.True(t, found)
	})
	t.Run("should remove expired keys", func(t *testing.T) {
		// given
		c := NewWithTimeout(100 * time.Millisecond)
		c.Set("dummy", "xxx", time.Millisecond*20)
		// when
		time.Sleep(time.Millisecond * 250)
		// then
		_, found := c.items.Load("dummy")
		assert.False(t, found)
	})
	t.Run("should not remove expired keys when closed", func(t *testing.T) {
		// given
		c := NewWithTimeout(100 * time.Millisecond)
		c.Set("dummy", "xxx", time.Millisecond*20)
		c.Close()
		// when
		time.Sleep(time.Millisecond * 250)
		// then
		_, found := c.items.Load("dummy")
		assert.True(t, found)
	})
}
