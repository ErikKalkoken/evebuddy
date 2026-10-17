package gamesearch

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestGameSearch_InitRecentItems(t *testing.T) {
	ids := func(ee []*app.EveEntity) []int64 {
		return xslices.Map(ee, func(x *app.EveEntity) int64 {
			return x.ID
		})
	}
	t.Run("prunes unknown IDs", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		existing := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{existing.ID, 99999999})
		a := NewGameSearch(u)

		a.init(t.Context())

		assert.Equal(t, []int64{existing.ID}, ids(a.recentItems))
		assert.Equal(t, []int64{existing.ID}, u.Settings().RecentSearches())
	})
	t.Run("keeps search picked before init finished", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		existing := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{existing.ID, 99999999})
		a := NewGameSearch(u)
		picked := factory.CreateEveEntity()
		a.recentItems = []*app.EveEntity{picked} // picked after init read the settings

		a.init(t.Context())

		assert.Equal(t, []int64{picked.ID, existing.ID}, ids(a.recentItems))
		assert.Equal(t, []int64{picked.ID, existing.ID}, u.Settings().RecentSearches())
	})
	t.Run("does not duplicate picked search", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{e1.ID, e2.ID})
		a := NewGameSearch(u)
		a.recentItems = []*app.EveEntity{e2}

		a.init(t.Context())

		assert.Equal(t, []int64{e2.ID, e1.ID}, ids(a.recentItems))
		assert.Equal(t, []int64{e2.ID, e1.ID}, u.Settings().RecentSearches())
	})
}
