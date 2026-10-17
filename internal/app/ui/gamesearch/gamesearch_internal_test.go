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

func TestGameSearch_Init(t *testing.T) {
	ids := func(ee []*app.EveEntity) []int64 {
		return xslices.Map(ee, func(x *app.EveEntity) int64 {
			return x.ID
		})
	}
	t.Run("disables search until recent searches are loaded", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})

		a := NewGameSearch(u)

		assert.True(t, a.searchEntry.Disabled())
		assert.Equal(t, "Loading...", a.searchEntry.PlaceHolder)
		assert.False(t, a.clearRecent.Visible())
		assert.False(t, a.resultsPage.Visible())
	})
	t.Run("enables search after loading", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		e := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)

		a.init(t.Context())

		assert.Equal(t, []int64{e.ID}, ids(a.recentItems))
		assert.False(t, a.searchEntry.Disabled())
		assert.Equal(t, "Search New Eden", a.searchEntry.PlaceHolder)
		assert.True(t, a.clearRecent.Visible())
	})
	t.Run("enables search when nothing was saved", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)

		a.init(t.Context())

		assert.Empty(t, a.recentItems)
		assert.False(t, a.searchEntry.Disabled())
		assert.Equal(t, "Search New Eden", a.searchEntry.PlaceHolder)
		assert.True(t, a.clearRecent.Visible())
	})
	t.Run("enables search when loading fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		e := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)
		if _, err := db.Exec("DROP TABLE eve_entities"); err != nil { // let loading fail
			t.Fatal(err)
		}

		a.init(t.Context())

		assert.Empty(t, a.recentItems)
		assert.Equal(t, []int64{e.ID}, u.Settings().RecentSearches())
		assert.False(t, a.searchEntry.Disabled())
		assert.Equal(t, "Search New Eden", a.searchEntry.PlaceHolder)
		assert.True(t, a.clearRecent.Visible())
	})
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
}
