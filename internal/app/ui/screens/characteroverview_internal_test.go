package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterOverviewFilter_Match(t *testing.T) {
	r := characterOverviewRow{
		alliance:        optional.New(&app.EveEntity{Name: "Goonswarm"}),
		corporation:     &app.EveEntity{Name: "Wayne Inc"},
		regionName:      "The Forge",
		solarSystemName: "Jita",
		tags:            set.Of("alpha"),
	}
	for _, tc := range []struct {
		name   string
		filter characterOverviewFilter
		want   bool
	}{
		{"no filter", characterOverviewFilter{}, true},
		{"alliance matches", characterOverviewFilter{alliance: "Goonswarm"}, true},
		{"alliance differs", characterOverviewFilter{alliance: "Test"}, false},
		{"corporation matches", characterOverviewFilter{corporation: "Wayne Inc"}, true},
		{"corporation differs", characterOverviewFilter{corporation: "Other"}, false},
		{"region differs", characterOverviewFilter{region: "Domain"}, false},
		{"system differs", characterOverviewFilter{solarSystem: "Amarr"}, false},
		{"tag matches", characterOverviewFilter{tag: "alpha"}, true},
		{"tag missing", characterOverviewFilter{tag: "bravo"}, false},
		{"all match", characterOverviewFilter{alliance: "Goonswarm", corporation: "Wayne Inc", region: "The Forge", solarSystem: "Jita", tag: "alpha"}, true},
		{"one of many differs", characterOverviewFilter{alliance: "Goonswarm", solarSystem: "Amarr"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(r))
		})
	}
}

func TestCharacterOverview_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []characterOverviewRow{
		{characterID: 1, characterName: "Bruce", corporation: &app.EveEntity{Name: "Wayne Inc"}, solarSystemName: "Jita", searchTarget: "bruce jita"},
		{characterID: 2, characterName: "Alice", corporation: &app.EveEntity{Name: "Other"}, solarSystemName: "Amarr", searchTarget: "alice amarr"},
	}
	newOverview := func(t *testing.T, isMobile bool) *CharacterOverview {
		a := NewCharacterOverview(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	characterIDs := func(a *CharacterOverview) []int64 {
		return xslices.Map(a.rowsFiltered, func(r characterOverviewRow) int64 {
			return r.characterID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		a.filterChip.SetSelected(map[string]string{characterOverviewFilterSystem: "Amarr"})
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newOverview(t, false)
		a.selectSolarSystem.SetSelected("Amarr")
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		a.searchEntry.SetText("bru")
		assert.ElementsMatch(t, []int64{1}, characterIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		assert.Equal(t, map[string]string{
			characterOverviewFilterAlliance:    "",
			characterOverviewFilterCorporation: "",
			characterOverviewFilterRegion:      "",
			characterOverviewFilterSystem:      "",
			characterOverviewFilterTag:         "",
		}, a.filterChip.Selected())
	})
}
