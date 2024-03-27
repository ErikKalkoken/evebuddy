package model_test

import (
	"example/esiapp/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Run("can set new value", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := model.CacheSet("dummy", []byte("xxx"), 100)
		// then
		assert.NoError(t, err)
	})
	t.Run("can get non expired value", func(t *testing.T) {
		// given
		model.TruncateTables()
		assert.NoError(t, model.CacheSet("dummy", []byte("xxx"), 100))
		// when
		o, err := model.CacheGet("dummy")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "xxx", string(o))
		}
	})
	t.Run("should return err when key has expired", func(t *testing.T) {
		// given
		model.TruncateTables()
		assert.NoError(t, model.CacheSet("dummy", []byte("xxx"), 0))
		// when
		o, err := model.CacheGet("dummy")
		// then
		assert.Equal(t, model.ErrCacheMiss, err)
		assert.Nil(t, o)
	})
	t.Run("should return error when timeout invalid", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := model.CacheSet("dummy", []byte("xxx"), -1)
		// then
		assert.Error(t, err)
	})
}

func TestCacheDelete(t *testing.T) {
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		model.TruncateTables()
		assert.NoError(t, model.CacheSet("dummy", []byte("test"), 100))
		// when
		err := model.CacheDelete("dummy")
		// then
		assert.NoError(t, err)
	})
	t.Run("should return cache miss error when key does not exit", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := model.CacheDelete("dummy")
		// then
		assert.Equal(t, model.ErrCacheMiss, err)
	})
}
