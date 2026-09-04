package pcache_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestPCache_Get(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should return value when found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		key := "key"
		value := []byte("value")
		c.Set(key, value, 0)

		// when
		got, found := c.Get(key)

		// then
		assert.True(t, found)
		xassert.Equal(t, value, got)
	})

	t.Run("should return value when found in storage and not memory", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		key := "key"
		value := []byte("value")
		err := st.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: time.Time{},
		})
		require.NoError(t, err)

		// when
		got, found := c.Get(key)

		// then
		assert.True(t, found)
		xassert.Equal(t, value, got)
	})

	t.Run("should return false when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()

		// when
		_, found := c.Get("key")
		assert.False(t, found)
	})

	t.Run("should not return expired keys", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := pcache.New(st, 0)
			defer c.Close()
			key := "key"
			value := []byte("value")
			c.Set(key, value, 5*time.Second)
			time.Sleep(6 * time.Second)
			synctest.Wait()

			// when
			_, found := c.Get(key)

			// then
			assert.False(t, found)
		})
	})

	t.Run("should not return expired keys from storage", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := pcache.New(st, 0)
			defer c.Close()
			key := "key"
			value := []byte("value")
			err := st.CacheSet(t.Context(), storage.CacheSetParams{
				Key:       key,
				Value:     value,
				ExpiresAt: time.Now().Add(5 * time.Second),
			})
			require.NoError(t, err)
			time.Sleep(6 * time.Second)
			synctest.Wait()

			// when
			_, found := c.Get(key)

			// then
			assert.False(t, found)
		})
	})
}

func TestPCache_Set(t *testing.T) {
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
		assert.True(t, found)
		xassert.Equal(t, value, got)
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
			xassert.Equal(t, value, got)
		}
	})
}

func TestPCache(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

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
		synctest.Test(t, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := pcache.New(st, 0)
			defer c.Close()
			c.Set("k1", []byte("dummy"), 10*time.Second)
			c.Set("k2", []byte("dummy"), 0)
			time.Sleep(15 * time.Second)
			synctest.Wait()

			// when
			got := c.CleanUp()

			// then
			assert.False(t, c.Exists("k1"))
			assert.True(t, c.Exists("k2"))
			xassert.Equal(t, 1, got)
		})
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

func TestPCache_Delete_StorageFailure(t *testing.T) {
	t.Run("should not remove entry from memory cache when storage delete fails", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBInMemory()
		c := pcache.New(st, 0)
		defer c.Close()
		key := "key"
		value := []byte("dummy")
		c.Set(key, value, 0)
		require.NoError(t, db.Close()) // force subsequent storage operations to fail

		// when
		c.Delete(key)

		// then
		got, found := c.Get(key)
		if assert.True(t, found, "expected entry to still be served from memory cache after failed storage delete") {
			xassert.Equal(t, value, got)
		}
	})
}

func TestPCache_Clear_StorageFailure(t *testing.T) {
	t.Run("should not remove entries from memory cache when storage clear fails", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBInMemory()
		c := pcache.New(st, 0)
		defer c.Close()
		key := "key"
		value := []byte("dummy")
		c.Set(key, value, 0)
		require.NoError(t, db.Close()) // force subsequent storage operations to fail

		// when
		c.Clear()

		// then
		got, found := c.Get(key)
		if assert.True(t, found, "expected entry to still be served from memory cache after failed storage clear") {
			xassert.Equal(t, value, got)
		}
	})
}
