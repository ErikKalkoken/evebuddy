package pcache_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPCache(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	t.Run("can set and get a cache entry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		value := []byte("value")
		// when
		c.Set("key", value, time.Minute)
		// then
		got, found := c.Get("key")
		if assert.True(t, found) {
			assert.Equal(t, value, got)
		}
	})
	t.Run("should create immortal cache", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		value := []byte("value")
		// when
		c.Set("key", value, 0)
		time.Sleep(250 * time.Millisecond)
		// then
		got, found := c.Get("key")
		if assert.True(t, found) {
			assert.Equal(t, value, got)
		}
	})
	t.Run("can check key existance 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		c.Set("key", []byte("dummy"), 0)
		// when
		assert.True(t, c.Exists("key"))
	})
	t.Run("can check key existance 2", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		// when
		assert.False(t, c.Exists("key"))
	})
	t.Run("can delete entry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		c.Set("key", []byte("dummy"), 0)
		// when
		c.Delete("key")
		// then
		assert.False(t, c.Exists("key"))
	})
	t.Run("can clear all entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		c.Set("k1", []byte("dummy"), 0)
		c.Set("k2", []byte("dummy"), 0)
		// when
		c.Clear()
		// then
		assert.False(t, c.Exists("k1"))
		assert.False(t, c.Exists("k2"))
	})
	t.Run("can clear expired entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		c.Set("k1", []byte("dummy"), time.Millisecond)
		c.Set("k2", []byte("dummy"), 0)
		time.Sleep(50 * time.Millisecond)
		// when
		got := c.CleanUp()
		// then
		assert.False(t, c.Exists("k1"))
		assert.True(t, c.Exists("k2"))
		assert.Equal(t, 1, got)
	})
	t.Run("can start with cleanup", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		c := pcache.New(st, 10*time.Minute)
		defer c.Close()
		// then
	})

}
