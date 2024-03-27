package cache_test

import (
	"example/esiapp/internal/cache"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Initialize the test database for this test package
	db, err := cache.InitDB(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	os.Exit(m.Run())
}

func TestGet(t *testing.T) {
	t.Run("can get non expired value", func(t *testing.T) {
		// given
		cache.Clear()
		assert.NoError(t, cache.Set("dummy", []byte("xxx"), 100))
		// when
		o, err := cache.Get("dummy")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "xxx", string(o))
		}
	})
	t.Run("should return err when key has expired", func(t *testing.T) {
		// given
		cache.Clear()
		assert.NoError(t, cache.Set("dummy", []byte("xxx"), 0))
		// when
		o, err := cache.Get("dummy")
		// then
		assert.Equal(t, cache.ErrCacheMiss, err)
		assert.Nil(t, o)
	})
}

func TestSet(t *testing.T) {
	t.Run("can set new value", func(t *testing.T) {
		// given
		cache.Clear()
		// when
		err := cache.Set("dummy", []byte("xxx"), 100)
		// then
		assert.NoError(t, err)
	})
	t.Run("should return error when timeout invalid", func(t *testing.T) {
		// given
		cache.Clear()
		// when
		err := cache.Set("dummy", []byte("xxx"), -1)
		// then
		assert.Error(t, err)
	})
}

func TestDelete(t *testing.T) {
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		cache.Clear()
		assert.NoError(t, cache.Set("dummy", []byte("test"), 100))
		// when
		err := cache.Delete("dummy")
		// then
		assert.NoError(t, err)
	})
	t.Run("should return cache miss error when key does not exit", func(t *testing.T) {
		// given
		cache.Clear()
		// when
		err := cache.Delete("dummy")
		// then
		assert.Equal(t, cache.ErrCacheMiss, err)
	})
}

func TestClear(t *testing.T) {
	t.Run("should clear the cache", func(t *testing.T) {
		// given
		cache.Clear()
		assert.NoError(t, cache.Set("dummy-1", []byte("test"), 100))
		assert.NoError(t, cache.Set("dummy-2", []byte("test"), 100))
		// when
		err := cache.Clear()
		// then
		assert.NoError(t, err)
		_, err = cache.Get("dummy-1")
		assert.Equal(t, cache.ErrCacheMiss, err)
		_, err = cache.Get("dummy-2")
		assert.Equal(t, cache.ErrCacheMiss, err)
	})
}
