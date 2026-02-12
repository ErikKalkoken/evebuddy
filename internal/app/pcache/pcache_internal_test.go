package pcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestPCache(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	t.Run("should create immortal cache on disk", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := New(st, 0)
		defer c.Close()
		key := "key"
		value := []byte("value")
		// when
		c.Set(key, value, 0)
		c.mc.Clear() // to ensure we access the DB entries
		time.Sleep(250 * time.Millisecond)
		// then
		x, found := c.Get(key)
		if assert.True(t, found) {
		xassert.Equal(t, value, x)
		}
	})
}
