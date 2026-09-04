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

func TestTimeoutFromExpiresAt(t *testing.T) {
	cases := []struct {
		name      string
		expiresAt time.Time
		wantOK    bool
	}{
		{"zero time means no expiry and should be cached", time.Time{}, true},
		{"a past expiry must not be cached as immortal", time.Now().Add(-time.Hour), false},
		{"an expiry at this exact instant must not be cached as immortal", time.Now(), false},
		{"a future expiry should be cached", time.Now().Add(time.Hour), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, ok := timeoutFromExpiresAt(c.expiresAt)
			assert.Equal(t, c.wantOK, ok)
		})
	}
}
