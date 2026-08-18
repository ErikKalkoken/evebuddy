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
