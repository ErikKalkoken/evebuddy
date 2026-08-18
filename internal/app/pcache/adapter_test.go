package pcache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestHTTPCacheAdapter(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	pc := pcache.New(st, 0)
	ca := pcache.NewHTTPCacheAdapter(pc, "prefix", 0)

	t.Run("get existing key", func(t *testing.T) {
		pc.Clear()
		ca.Set("a", []byte("alpha"))
		got, ok := ca.Get("a")
		if assert.True(t, ok) {
			assert.Equal(t, []byte("alpha"), got)
		}
	})

	t.Run("get non existing key", func(t *testing.T) {
		pc.Clear()
		_, ok := ca.Get("a")
		assert.False(t, ok)
	})

	t.Run("delete existing key", func(t *testing.T) {
		pc.Clear()
		ca.Set("a", []byte("alpha"))
		ca.Delete("a")
		_, ok := ca.Get("a")
		assert.False(t, ok)
	})
}

func TestServiceCacheAdapter(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	pc := pcache.New(st, 0)
	sa := pcache.NewServiceCacheAdapter(pc, "svc")

	t.Run("string operations", func(t *testing.T) {
		pc.Clear()
		sa.SetString("k1", "val1", 0)
		got, ok := sa.GetString("k1")
		if assert.True(t, ok) {
			assert.Equal(t, "val1", got)
		}

		sa.Delete("k1")
		_, ok = sa.GetString("k1")
		assert.False(t, ok)
	})

	t.Run("int64 operations", func(t *testing.T) {
		pc.Clear()
		var expected int64 = 9223372036854775807
		sa.SetInt64("k2", expected, 0)
		got, ok := sa.GetInt64("k2")
		if assert.True(t, ok) {
			assert.Equal(t, expected, got)
		}
	})

	t.Run("get non-existing key", func(t *testing.T) {
		pc.Clear()
		_, okStr := sa.GetString("missing")
		_, okInt := sa.GetInt64("missing")
		assert.False(t, okStr)
		assert.False(t, okInt)
	})

	t.Run("get corrupted int64 payload", func(t *testing.T) {
		pc.Clear()
		pc.Set("svc:short", []byte{1, 2, 3}, 0)
		_, ok := sa.GetInt64("short")
		assert.False(t, ok)
	})
}
