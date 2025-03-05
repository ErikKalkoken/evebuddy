package pcache_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPCache(t *testing.T) {
	db, st, _ := testutil.New()
	defer db.Close()
	t.Run("can set and get a cache entry", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		key := "key"
		value := []byte("value")
		// when
		c.Set(key, value, time.Minute)
		// then
		x, found := c.Get(key)
		if assert.True(t, found) {
			assert.Equal(t, value, x)
		}
	})
	t.Run("should create immortal cache", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := pcache.New(st, 0)
		defer c.Close()
		key := "key"
		value := []byte("value")
		// when
		c.Set(key, value, 0)
		time.Sleep(250 * time.Millisecond)
		// then
		x, found := c.Get(key)
		if assert.True(t, found) {
			assert.Equal(t, value, x)
		}
	})
}
