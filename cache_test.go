package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestCacheAdapter2(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	pc := pcache.New(st, 0)
	ca := newServiceCacheAdapter(pc, "prefix")
	t.Run("get existing key", func(t *testing.T) {
		pc.Clear()
		v := int64(42)
		ca.SetInt64("a", v, 0)
		got, ok := ca.GetInt64("a")
		if assert.True(t, ok) {
			assert.Equal(t, v, got)
		}
	})
	t.Run("get non existing key", func(t *testing.T) {
		pc.Clear()
		_, ok := ca.GetInt64("a")
		assert.False(t, ok)
	})
}
